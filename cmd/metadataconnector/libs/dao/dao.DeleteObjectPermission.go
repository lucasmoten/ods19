package dao

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
)

// DeleteObjectPermission uses the passed in objectPermission and makes the
// appropriate sql calls to the database to validate that the token is current,
// and is not yet deleted before marking the permission as deleted
//    objectPermission.ID must be set to the permission to be marked as deleted
//    objectPermission.ChangeToken must be set to the current value
//    objectPermission.ModifiedBy must be set to the user performing the operation
func (dao *DataAccessLayer) DeleteObjectPermission(objectPermission models.ODObjectPermission, propagateToChildren bool) (models.ODObjectPermission, error) {

	tx := dao.MetadataDB.MustBegin()
	dbObjectPermission, err := deleteObjectPermissionInTransaction(tx, objectPermission, propagateToChildren)
	if err != nil {
		log.Printf("Error in DeleteObjectPermission: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectPermission, err
}

func deleteObjectPermissionInTransaction(tx *sqlx.Tx, objectPermission models.ODObjectPermission, propagateToChildren bool) (models.ODObjectPermission, error) {
	dbObjectPermission := models.ODObjectPermission{}
	if objectPermission.ID == nil {
		return dbObjectPermission, ErrMissingID
	}
	if objectPermission.ChangeToken == "" {
		return dbObjectPermission, ErrMissingChangeToken
	}
	// Fetch object permission
	dbObjectPermission, err := getObjectPermissionInTransaction(tx, objectPermission)
	if err != nil {
		return dbObjectPermission, err
	}
	// Check if changeToken matches
	if objectPermission.ChangeToken != dbObjectPermission.ChangeToken {
		return dbObjectPermission, fmt.Errorf("ChangeToken does not match expected value %s", dbObjectPermission.ChangeToken)
	}
	// Check if already deleted
	if dbObjectPermission.IsDeleted {
		// NOOP
		return dbObjectPermission, nil
	}
	// Mark property as deleted
	dbObjectPermission.IsDeleted = true
	dbObjectPermission.ModifiedBy = objectPermission.ModifiedBy
	dbObjectPermission.DeletedBy.String = objectPermission.ModifiedBy
	dbObjectPermission.DeletedBy.Valid = true
	updateObjectPermissionStatement, err := tx.Preparex(
		`update object_permission set modifiedby = ?, isdeleted = ?, deletedby = ? where id = ?`)
	if err != nil {
		return dbObjectPermission, err
	}
	_, err = updateObjectPermissionStatement.Exec(dbObjectPermission.ModifiedBy, dbObjectPermission.IsDeleted, dbObjectPermission.DeletedBy.String, dbObjectPermission.ID)
	if err != nil {
		return dbObjectPermission, err
	}

	// Refetch to pick up changed state for deleted date
	dbObjectPermission, err = getObjectPermissionInTransaction(tx, objectPermission)
	if err != nil {
		return dbObjectPermission, err
	}

	// Do Recursively to children
	if propagateToChildren {
		// Find matching inherited permissions for children of the object for
		// which the share was just deleted that are not explicit permissions
		matchingPermission := []models.ODObjectPermission{}
		query := `select op.* from object_permission op 
            inner join object o on op.objectid = o.id 
            where op.isdeleted = 0 and op.explicitshare = 0 and o.isdeleted = 0
                and o.parentid = ? and op.allowcreate = ? and op.allowread = ? 
                and op.allowupdate = ? and op.allowdelete = ? and op.allowshare = ?
                and op.Grantee = ?`
		err := tx.Select(&matchingPermission, query, dbObjectPermission.ObjectID,
			dbObjectPermission.AllowCreate, dbObjectPermission.AllowRead,
			dbObjectPermission.AllowUpdate, dbObjectPermission.AllowDelete,
			dbObjectPermission.AllowShare, dbObjectPermission.Grantee)
		if err != nil {
			return dbObjectPermission, err
		}
		for _, childPermission := range matchingPermission {
			childPermission.ModifiedBy = objectPermission.ModifiedBy
			_, err := deleteObjectPermissionInTransaction(tx, childPermission, propagateToChildren)
			if err != nil {
				return dbObjectPermission, err
			}
		}
	}

	return dbObjectPermission, nil
}

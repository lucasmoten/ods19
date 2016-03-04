package dao

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
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
		return dbObjectPermission, errMissingID
	}
	if objectPermission.ChangeToken == "" {
		return dbObjectPermission, errMissingChangeToken
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
	updateObjectPermissionStatement, err := tx.Preparex(
		`update object_permission set modifiedby = ?, isdeleted = ?, deletedby = ? where id = ?`)
	if err != nil {
		return dbObjectPermission, err
	}
	_, err = updateObjectPermissionStatement.Exec(dbObjectPermission.ModifiedBy, dbObjectPermission.IsDeleted, dbObjectPermission.ModifiedBy, dbObjectPermission.ID)
	if err != nil {
		return dbObjectPermission, err
	}

	// Refetch to pick up changed state for deleted date
	dbObjectPermission, err = getObjectPermissionInTransaction(tx, objectPermission)
	if err != nil {
		return dbObjectPermission, err
	}

	// Do Recursively to children?
	if propagateToChildren {
		// TODO: Determine logic for this as it needs to account for whether this is deleting...
		//  - all permission other then those establishing full permissions to the owner
		//  - only permissions created by the same person deleting this one (objectPermission.ModifiedBy)
		//  - only permissions matching the same share settings as that configured passed in
		//  - only permissions matching the same share settings and created by the person deleting
		// lots of options. need to discuss
	}

	return dbObjectPermission, nil
}

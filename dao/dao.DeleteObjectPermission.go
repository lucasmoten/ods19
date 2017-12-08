package dao

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
)

// DeleteObjectPermission uses the passed in objectPermission and makes the
// appropriate sql calls to the database to validate that the token is current,
// and is not yet deleted before marking the permission as deleted
//    objectPermission.ID must be set to the permission to be marked as deleted
//    objectPermission.ChangeToken must be set to the current value
//    objectPermission.ModifiedBy must be set to the user performing the operation
func (dao *DataAccessLayer) DeleteObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
	defer util.Time("DeleteObjectPermission")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectPermission{}, err
	}
	dbObjectPermission, err := deleteObjectPermissionInTransaction(tx, objectPermission)
	if err != nil {
		dao.GetLogger().Error("Error in DeleteObjectPermission", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectPermission, err
}

func deleteObjectPermissionInTransaction(tx *sqlx.Tx, objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
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
	// Check if already deleted
	if dbObjectPermission.IsDeleted {
		// NOOP
		return dbObjectPermission, nil
	}
	// Check if changeToken matches
	if objectPermission.ChangeToken != dbObjectPermission.ChangeToken {
		return dbObjectPermission, fmt.Errorf("ChangeToken does not match expected value %s", dbObjectPermission.ChangeToken)
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

	return dbObjectPermission, nil
}

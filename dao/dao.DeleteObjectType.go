package dao

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
)

// DeleteObjectType uses the passed in objectType and makes the appropriate sql
// calls to the database to validate that the token is current, and is not yet
// deleted before marking the object type as deleted
//    objectType.ID must be set to the objectType to be marked as deleted
//    objectType.ChangeToken must be set to the current value
//    objectType.ModifiedBy must be set to the user performing the operation
func (dao *DataAccessLayer) DeleteObjectType(objectType models.ODObjectType) error {
	defer util.Time("DeleteObjectPropertyType")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return err
	}
	err = deleteObjectTypeInTransaction(tx, objectType)
	if err != nil {
		dao.GetLogger().Error("Error in DeleteObjectType", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func deleteObjectTypeInTransaction(tx *sqlx.Tx, objectType models.ODObjectType) error {
	// Pre-DB Validation
	if objectType.ID == nil {
		return ErrMissingID
	}
	if objectType.ChangeToken == "" {
		return ErrMissingChangeToken
	}
	existingObjectType, err := getObjectTypeInTransaction(tx, objectType)
	if err != nil {
		return err
	}
	if objectType.ChangeToken != existingObjectType.ChangeToken {
		return fmt.Errorf("ChangeToken does not match expected value %s", existingObjectType.ChangeToken)
	}
	// Check if already deleted
	if existingObjectType.IsDeleted {
		// NOOP
		return nil
	}
	// Mark as deleted
	existingObjectType.IsDeleted = true
	existingObjectType.ModifiedBy = objectType.ModifiedBy
	updateObjectTypeStatement, err := tx.Preparex(
		`update object_type set modifiedby = ?, isdeleted = ? where id = ?`)
	if err != nil {
		return err
	}
	defer updateObjectTypeStatement.Close()
	_, err = updateObjectTypeStatement.Exec(
		existingObjectType.ModifiedBy, existingObjectType.IsDeleted, existingObjectType.ID)
	if err != nil {
		return err
	}
	return nil
}

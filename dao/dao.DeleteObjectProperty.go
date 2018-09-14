package dao

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// DeleteObjectProperty uses the passed in objectProperty and makes the
// appropriate sql calls to the database to validate that the token is current,
// and is not yet deleted before marking the property as deleted along with the
// associated object_property relationship
//    objectProperty.ID must be set to the property to be marked as deleted
//    objectProperty.ChangeToken must be set to the current value
//    objectProperty.ModifiedBy must be set to the user performing the operation
func (dao *DataAccessLayer) DeleteObjectProperty(objectProperty models.ODObjectPropertyEx) error {
	defer util.Time("DeleteObjectProperty")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return err
	}
	err = deleteObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		dao.GetLogger().Error("Error in DeleteObjectProperty", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func deleteObjectPropertyInTransaction(tx *sqlx.Tx, objectProperty models.ODObjectPropertyEx) error {
	if objectProperty.ID == nil {
		return ErrMissingID
	}
	if objectProperty.ChangeToken == "" {
		return ErrMissingChangeToken
	}
	// Fetch object property
	dbObjectProperty, err := getObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		return err
	}
	// // Check if changeToken matches
	// if objectProperty.ChangeToken != dbObjectProperty.ChangeToken {
	// 	return fmt.Errorf("ChangeToken %s for property %s does not match expected value %s", objectProperty.ChangeToken, objectProperty.Name, dbObjectProperty.ChangeToken)
	// }
	// Check if already deleted
	if dbObjectProperty.IsDeleted {
		// NOOP
		return nil
	}
	// Mark property as deleted
	dbObjectProperty.IsDeleted = true
	dbObjectProperty.ModifiedBy = objectProperty.ModifiedBy
	updateObjectPropertyStatement, err := tx.Preparex(
		`update property set modifiedby = ?, isdeleted = ? where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectPropertyStatement.Exec(dbObjectProperty.ModifiedBy, dbObjectProperty.IsDeleted, dbObjectProperty.ID)
	if err != nil {
		return err
	}
	updateObjectPropertyStatement.Close()
	// Mark relationship between the property and objects as deleted
	updateRelationshipStatement, err := tx.Preparex(
		`update object_property set modifiedby = ?, isdeleted = ? where propertyid = ?`)
	if err != nil {
		return err
	}
	defer updateRelationshipStatement.Close()
	_, err = updateRelationshipStatement.Exec(dbObjectProperty.ModifiedBy, dbObjectProperty.IsDeleted, dbObjectProperty.ID)
	if err != nil {
		return err
	}
	return nil
}

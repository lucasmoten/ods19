package dao

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// DeleteObjectProperty uses the passed in objectProperty and makes the
// appropriate sql calls to the database to validate that the token is current,
// and is not yet deleted before marking the property as deleted along with the
// associated object_property relationship
//    objectProperty.ID must be set to the property to be marked as deleted
//    objectProperty.ChangeToken must be set to the current value
//    objectProperty.ModifiedBy must be set to the user performing the operation
func (dao *DataAccessLayer) DeleteObjectProperty(objectProperty *models.ODObjectPropertyEx) error {

	tx := dao.MetadataDB.MustBegin()
	err := deleteObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func deleteObjectPropertyInTransaction(tx *sqlx.Tx, objectProperty *models.ODObjectPropertyEx) error {
	if objectProperty.ID == nil {
		return errMissingID
	}
	if objectProperty.ChangeToken == "" {
		return errMissingChangeToken
	}
	// Fetch object property
	dbObjectProperty, err := getObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		return err
	}
	// Check if changeToken matches
	if objectProperty.ChangeToken != dbObjectProperty.ChangeToken {
		return fmt.Errorf("ChangeToken does not match expected value %s", dbObjectProperty.ChangeToken)
	}
	// Check if already deleted
	if dbObjectProperty.IsDeleted {
		// NOOP
		return nil
	}
	// Mark property as deleted
	dbObjectProperty.IsDeleted = true
	dbObjectProperty.ModifiedBy = objectProperty.ModifiedBy
	updateObjectPropertyStatement, err := tx.Prepare(
		`update property set modifiedby = ?, isdeleted = ? where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectPropertyStatement.Exec(dbObjectProperty.ModifiedBy, dbObjectProperty.IsDeleted, dbObjectProperty.ID)
	if err != nil {
		return err
	}
	// Mark relationship between the property and objects as deleted
	updateRelationshipStatement, err := tx.Prepare(
		`update object_property set modifiedby = ?, isdeleted = ? where propertyid = ?`)
	if err != nil {
		return err
	}
	_, err = updateRelationshipStatement.Exec(dbObjectProperty.ModifiedBy, dbObjectProperty.IsDeleted, dbObjectProperty.ID)
	if err != nil {
		return err
	}
	// TODO: Anything else need deleted based on this object type ?

	return nil
}

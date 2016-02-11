package dao

import (
	"errors"
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// UpdateObjectProperty uses the passed in objectProperty and makes the
// appropriate sql calls to the database to validate that the token is current,
// and is not yet deleted before making changes to the property value and its
// classification portion mark
//    objectProperty.ID must be set to the property to be updated
//    objectProperty.ChangeToken must be set to the current value
//    objectProperty.ModifiedBy must be set to the user performing the operation
//    objectProperty.Value.String must be set to the new value.
func UpdateObjectProperty(db *sqlx.DB, objectProperty *models.ODObjectPropertyEx) error {
	// Pre-DB Validation
	if objectProperty.ID == nil {
		return errors.New("ID was not specified for object property being updated")
	}
	if objectProperty.ChangeToken == "" {
		return errors.New("ChangeToken was not specified for object property being updated")
	}
	// Fetch object property
	dbObjectProperty, err := GetObjectProperty(db, objectProperty)
	if err != nil {
		return err
	}
	// Check if changeToken matches
	if objectProperty.ChangeToken != dbObjectProperty.ChangeToken {
		return fmt.Errorf("ChangeToken does not match expected value %s", dbObjectProperty.ChangeToken)
	}
	// Check if deleted
	if dbObjectProperty.IsDeleted {
		// NOOP
		return nil
	}
	// Setup property
	dbObjectProperty.ModifiedBy = objectProperty.ModifiedBy
	dbObjectProperty.Value.String = objectProperty.Value.String
	dbObjectProperty.ClassificationPM.String = objectProperty.ClassificationPM.String
	updateObjectPropertyStatement, err := db.Prepare(`update property set modifiedby = ?, value = ?, classificationpm = ? where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectPropertyStatement.Exec(dbObjectProperty.ModifiedBy, dbObjectProperty.Value.String, dbObjectProperty.ClassificationPM.String, dbObjectProperty.IsDeleted, dbObjectProperty.ID)
	if err != nil {
		return err
	}

	return nil
}

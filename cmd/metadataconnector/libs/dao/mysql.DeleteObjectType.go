package dao

import (
	"errors"
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// DeleteObjectType uses the passed in objectType and makes the appropriate sql
// calls to the database to validate that the token is current, and is not yet
// deleted before marking the object type as deleted
//    objectType.ID must be set to the objectType to be marked as deleted
//    objectType.ChangeToken must be set to the current value
//    objectType.ModifiedBy must be set to the user performing the operation
func DeleteObjectType(db *sqlx.DB, objectType *models.ODObjectType) error {
	// Pre-DB Validation
	if objectType.ID == nil {
		return errors.New("ID was not specified for object type being deleted")
	}
	if objectType.ChangeToken == "" {
		return errors.New("ChangeToken was not specified for object type being deleted")
	}

	// Fetch object type
	dbObjectType, err := GetObjectType(db, objectType)
	if err != nil {
		return err
	}
	// Check if changeToken matches
	if objectType.ChangeToken != dbObjectType.ChangeToken {
		return fmt.Errorf("ChangeToken does not match expected value %s", dbObjectType.ChangeToken)
	}
	// Check if already deleted
	if dbObjectType.IsDeleted {
		// NOOP
		return nil
	}

	// Mark as deleted
	dbObjectType.IsDeleted = true
	dbObjectType.ModifiedBy = objectType.ModifiedBy
	updateObjectTypeStatement, err := db.Prepare(`update object_type set modifiedby = ?, isdeleted = ? where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectTypeStatement.Exec(dbObjectType.ModifiedBy, dbObjectType.IsDeleted, dbObjectType.ID)
	if err != nil {
		return err
	}

	// TODO: Anything else need deleted based on this object type ?

	return nil
}

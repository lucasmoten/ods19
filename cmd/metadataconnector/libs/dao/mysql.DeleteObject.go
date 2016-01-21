package dao

import (
	"errors"
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// DeleteObject uses the passed in object and makes the appropriate sql calls to
// the database to validate that the token is current, and is not yet deleted
// before marking the object as deleted and marking descendents deleted as well
//    object.ID must be set to the object to be marked as deleted
//    object.ChangeToken must be set to the current value
//    object.ModifiedBy must be set to the user performing the operation
//    explicit denotes whether this object will be marked IsDeleted and
//      IsAncestorDeleted. IsAncestorDeleted is only set if explicit = false
//      whose purpose is to mark child items as implicitly deleted due to an
//      ancestor being deleted.
func DeleteObject(db *sqlx.DB, object *models.ODObject, explicit bool) error {

	// Pre-DB Validation
	if object.ID == nil {
		return errors.New("Object ID was not specified for object being deleted")
	}
	if object.ChangeToken == "" {
		return errors.New("Object ChangeToken was not specified for object being deleted")
	}

	// Fetch object
	dbObject, err := GetObject(db, object, false)
	if err != nil {
		return err
	}
	// Check if changeToken matches
	if object.ChangeToken != dbObject.ChangeToken {
		return fmt.Errorf("Object ChangeToken does not match expected value %s", dbObject.ChangeToken)
	}
	// Check if already deleted
	if dbObject.IsDeleted {
		// NOOP
		return nil
	}

	// Mark as deleted
	dbObject.IsDeleted = true
	dbObject.ModifiedBy = object.ModifiedBy
	dbObject.IsAncestorDeleted = !explicit
	updateObjectStatement, err := db.Prepare(`update object set modifiedby = ?, isdeleted = ?, isancestordeleted = ? where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectStatement.Exec(dbObject.ModifiedBy, dbObject.IsDeleted, dbObject.IsAncestorDeleted, dbObject.ID)
	if err != nil {
		return err
	}

	// TODO: Anything else need deleted based on this object?

	// Process children
	resultset, err := GetChildObjects(db, "", 1, 10000, dbObject)
	for i := 0; i < len(resultset.Objects); i++ {
		err = DeleteObject(db, &resultset.Objects[i], false)
		if err != nil {
			return err
		}
	}

	return nil
}

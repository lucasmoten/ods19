package dao

import (
	"errors"
	"fmt"
	"time"

	"decipher.com/oduploader/metadata/models"
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
func (dao *DataAccessLayer) DeleteObject(object *models.ODObject, explicit bool) error {

	// Pre-DB Validation
	if object.ID == nil {
		return errors.New("Object ID was not specified for object being deleted")
	}
	if object.ChangeToken == "" {
		return errors.New("Object ChangeToken was not specified for object being deleted")
	}

	// Fetch object
	dbObject, err := dao.GetObject(object, false)
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
	dbObject.DeletedDate.Time = time.Now().UTC()
	dbObject.DeletedDate.Valid = true
	dbObject.IsAncestorDeleted = !explicit
	updateObjectStatement, err := dao.MetadataDB.Prepare(`
    update object set modifiedby = ?, isdeleted = ?, deletedDate = ?, isancestordeleted = ? where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectStatement.Exec(dbObject.ModifiedBy, dbObject.IsDeleted, dbObject.DeletedDate, dbObject.IsAncestorDeleted, dbObject.ID)
	if err != nil {
		return err
	}

	// TODO: Anything else need deleted based on this object?

	// Process children
	resultset, err := dao.GetChildObjects("", 1, 10000, dbObject)
	for i := 0; i < len(resultset.Objects); i++ {
		err = dao.DeleteObject(&resultset.Objects[i], false)
		if err != nil {
			return err
		}
	}

	return nil
}

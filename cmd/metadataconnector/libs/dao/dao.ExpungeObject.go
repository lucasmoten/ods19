package dao

import (
	"errors"
	"fmt"
	"time"

	"decipher.com/oduploader/metadata/models"
)

// ExpungeObject uses the passed in object and makes the appropriate sql calls
// to the database to validate that the token is current, and is not yet
// expunged before marking the object as deleted and expunged and marking its
// descendents deleted as well
//    object.ID must be set to the object to be marked as deleted
//    object.ChangeToken must be set to the current value
//    object.ModifiedBy must be set to the user performing the operation
//    explicit denotes whether this object will be marked IsDeleted and
//      IsAncestorDeleted. IsAncestorDeleted is only set if explicit = false
//      whose purpose is to mark child items as implicitly deleted due to an
//      ancestor being deleted.
func (dao *DataAccessLayer) ExpungeObject(object *models.ODObject, explicit bool) error {

	// Pre-DB Validation
	if object.ID == nil {
		return errors.New("Object ID was not specified for object being expunged")
	}
	if object.ChangeToken == "" {
		return errors.New("Object ChangeToken was not specified for object being expunged")
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
	// Check if already expunged
	if dbObject.IsExpunged {
		// NOOP
		return nil
	}

	// Mark as deleted and expunged
	actionTime := time.Now().UTC()
	dbObject.ModifiedBy = object.ModifiedBy
	if !dbObject.IsDeleted {
		dbObject.IsDeleted = true
		dbObject.DeletedDate.Time = actionTime
		dbObject.DeletedDate.Valid = true
		dbObject.DeletedBy.String = dbObject.ModifiedBy
		dbObject.DeletedBy.Valid = true
	}
	dbObject.IsAncestorDeleted = !explicit
	dbObject.IsExpunged = true
	dbObject.ExpungedDate.Time = actionTime
	dbObject.ExpungedDate.Valid = true
	dbObject.ExpungedBy.String = dbObject.ModifiedBy
	dbObject.ExpungedBy.Valid = true
	updateObjectStatement, err := dao.MetadataDB.Prepare(`
    update object set modifiedby = ?,
    isdeleted = ?, deleteddate = ?, deletedby = ?,
    isancestordeleted = ?,
    isexpunged = ?, expungeddate = ?, expungedby = ?
    where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectStatement.Exec(dbObject.ModifiedBy,
		dbObject.IsDeleted, dbObject.DeletedDate, dbObject.DeletedBy,
		dbObject.IsAncestorDeleted,
		dbObject.IsExpunged, dbObject.ExpungedDate, dbObject.ExpungedBy,
		dbObject.ID)
	if err != nil {
		return err
	}

	// Process children
	resultset, err := dao.GetChildObjects("", 1, 10000, dbObject)
	for i := 0; i < len(resultset.Objects); i++ {
		err = dao.ExpungeObject(&resultset.Objects[i], false)
		if err != nil {
			return err
		}
	}

	return nil
}

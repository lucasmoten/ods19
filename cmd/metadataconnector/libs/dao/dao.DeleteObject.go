package dao

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"

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
func (dao *DataAccessLayer) DeleteObject(object models.ODObject, explicit bool) error {
	tx := dao.MetadataDB.MustBegin()
	err := deleteObjectInTransaction(tx, object, explicit)
	if err != nil {
		log.Printf("Error in DeleteObject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func deleteObjectInTransaction(tx *sqlx.Tx, object models.ODObject, explicit bool) error {
	// Pre-DB Validation
	if object.ID == nil {
		return errors.New("Object ID was not specified for object being deleted")
	}
	if object.ChangeToken == "" {
		return errors.New("Object ChangeToken was not specified for object being deleted")
	}

	// Fetch object
	dbObject, err := getObjectInTransaction(tx, object, false)
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
	actionTime := time.Now().UTC()
	dbObject.ModifiedBy = object.ModifiedBy
	dbObject.IsDeleted = true
	dbObject.DeletedDate.Time = actionTime
	dbObject.DeletedDate.Valid = true
	dbObject.DeletedBy.String = dbObject.ModifiedBy
	dbObject.DeletedBy.Valid = true
	dbObject.IsAncestorDeleted = !explicit
	updateObjectStatement, err := tx.Preparex(`
    update object set modifiedby = ?,
		isdeleted = ?, deleteddate = ?, deletedby = ?,
		isancestordeleted = ? where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectStatement.Exec(dbObject.ModifiedBy,
		dbObject.IsDeleted, dbObject.DeletedDate, dbObject.DeletedBy,
		dbObject.IsAncestorDeleted, dbObject.ID)
	if err != nil {
		return err
	}

	// Process children
	resultset, err := getChildObjectsInTransaction(tx, "", 1, 10000, dbObject)
	for i := 0; i < len(resultset.Objects); i++ {
		if !resultset.Objects[i].IsAncestorDeleted {
			authorizedToDelete := false
			for _, permission := range resultset.Objects[i].Permissions {
				if permission.Grantee == object.ModifiedBy &&
					permission.AllowDelete {
					authorizedToDelete = true
					break
				}
			}
			if authorizedToDelete {
				resultset.Objects[i].ModifiedBy = object.ModifiedBy
				err = deleteObjectInTransaction(tx, resultset.Objects[i], false)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

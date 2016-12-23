package dao

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
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
func (dao *DataAccessLayer) ExpungeObject(user models.ODUser, object models.ODObject, explicit bool) error {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	object.ModifiedBy = user.DistinguishedName

	updateObjectStatement, err := expungeObjectInTransactionPrepare(tx)
	defer updateObjectStatement.Close()

	err = expungeObjectInTransaction(tx, user, object, explicit, updateObjectStatement)
	if err != nil {
		dao.GetLogger().Error("Error in ExpungeObject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func expungeObjectInTransactionPrepare(tx *sqlx.Tx) (*sqlx.Stmt, error) {
	return tx.Preparex(`
    update object set modifiedby = ?,
    isdeleted = ?, deleteddate = ?, deletedby = ?,
    isancestordeleted = ?,
    isexpunged = ?, expungeddate = ?, expungedby = ?
    where id = ?`)
}

func expungeObjectInTransaction(tx *sqlx.Tx, user models.ODUser, object models.ODObject, explicit bool, updateObjectStatement *sqlx.Stmt) error {
	// Pre-DB Validation
	if object.ID == nil {
		return errors.New("Object ID was not specified for object being expunged")
	}
	if object.ChangeToken == "" {
		return errors.New("Object ChangeToken was not specified for object being expunged")
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

	_, err = updateObjectStatement.Exec(dbObject.ModifiedBy,
		dbObject.IsDeleted, dbObject.DeletedDate, dbObject.DeletedBy,
		dbObject.IsAncestorDeleted,
		dbObject.IsExpunged, dbObject.ExpungedDate, dbObject.ExpungedBy,
		dbObject.ID)
	if err != nil {
		return err
	}

	// Process children
	hasUndeletedChildren := true
	deletedAtLeastOne := true
	pagingRequest := PagingRequest{PageNumber: 1, PageSize: MaxPageSize}
	for hasUndeletedChildren {
		pagedResultset, err := getChildObjectsInTransaction(tx, pagingRequest, dbObject, false)
		hasUndeletedChildren = (pagedResultset.PageCount > pagingRequest.PageNumber) && deletedAtLeastOne
		for i := 0; i < len(pagedResultset.Objects); i++ {
			deletedAtLeastOne = false
			if !pagedResultset.Objects[i].IsAncestorDeleted {
				authorizedToDelete := false
				for _, permission := range pagedResultset.Objects[i].Permissions {
					if permission.AllowDelete && isUserMemberOf(user, permission.Grantee) {
						authorizedToDelete = true
						break
					}
				}
				if authorizedToDelete {
					pagedResultset.Objects[i].ModifiedBy = object.ModifiedBy
					err = expungeObjectInTransaction(tx, user, pagedResultset.Objects[i], false, updateObjectStatement)
					if err != nil {
						return err
					}
					deletedAtLeastOne = true
				}
			}
		}
		if !deletedAtLeastOne {
			pagingRequest.PageNumber++
		}
	}

	return nil
}

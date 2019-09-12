package dao

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models/acm"
	"bitbucket.di2e.net/dime/object-drive-server/util"
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
func (dao *DataAccessLayer) DeleteObject(user models.ODUser, object models.ODObject, explicit bool) error {
	defer util.Time("DeleteObject")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return err
	}
	err = deleteObjectInTransaction(tx, user, object, explicit)
	if err != nil {
		dao.GetLogger().Error("Error in DeleteObject", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func deleteObjectInTransaction(tx *sqlx.Tx, user models.ODUser, object models.ODObject, explicit bool) error {
	// Pre-DB Validation
	if object.ID == nil {
		return errors.New("Object ID was not specified for object being deleted")
	}
	if object.ChangeToken == "" {
		return errors.New("Object ChangeToken was not specified for object being deleted")
	}

	// Fetch object
	dbObject, err := getObjectInTransaction(tx, object, false, false)
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

	// Populate user snippets from database
	if explicit {
		user.Snippets, err = getUserSnippets(tx, user)
		if err != nil {
			return err
		}
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
	updateObjectStatement, err := tx.Preparex(`update object set 
        modifiedBy = ?
        ,isDeleted = ?
        ,deletedDate = ?
        ,deletedBy = ?
        ,isAncestorDeleted = ? 
    where id = ?`)
	if err != nil {
		return err
	}
	defer updateObjectStatement.Close()
	_, err = updateObjectStatement.Exec(dbObject.ModifiedBy,
		dbObject.IsDeleted, dbObject.DeletedDate, dbObject.DeletedBy,
		dbObject.IsAncestorDeleted, dbObject.ID)
	if err != nil {
		return err
	}
	// Process children
	hasUndeletedChildren := true
	deletedAtLeastOne := true
	pagingRequest := PagingRequest{PageNumber: 1, PageSize: 100}
	for hasUndeletedChildren {
		pagedResultset, err := getChildObjectsInTransaction(tx, pagingRequest, dbObject, true, false)
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
					err = deleteObjectInTransaction(tx, user, pagedResultset.Objects[i], false)
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

func isUserMemberOf(user models.ODUser, groupName string) bool {
	// if called without snippets, then user has no access
	if user.Snippets == nil {
		return false
	}

	// find the relevant snippet
	var shareSnippet acm.RawSnippetFields
	for _, rawFields := range user.Snippets.Snippets {
		switch rawFields.FieldName {
		case "f_share":
			shareSnippet = rawFields
			break
		default:
			continue
		}
	}

	// snippet portion without values implies ZERO membership in any groups, even self!!!
	if len(shareSnippet.Values) == 0 {
		return false
	}

	// iterate the values
	for _, shareValue := range shareSnippet.Values {
		// case insensitive check
		if strings.Compare(strings.ToLower(shareValue), strings.ToLower(groupName)) == 0 {
			return true
		}
	}

	// no matches
	return false

}

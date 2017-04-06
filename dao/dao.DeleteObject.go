package dao

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
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
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	err = deleteObjectInTransaction(tx, user, object, explicit)
	if err != nil {
		dao.GetLogger().Error("Error in DeleteObject", zap.String("err", err.Error()))
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

	// Option to populate user snippets from database
	if explicit && isOption409() {
		user.Snippets, err = getUserSnippets(tx, user)
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
	_, err = updateObjectStatement.Exec(dbObject.ModifiedBy,
		dbObject.IsDeleted, dbObject.DeletedDate, dbObject.DeletedBy,
		dbObject.IsAncestorDeleted, dbObject.ID)
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

func isOption409() bool {
	option409 := os.Getenv("OD_OPTION_409")
	option409 = strings.ToLower(strings.TrimSpace(option409))
	return option409 == "true"
}

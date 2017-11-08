package dao

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"

	"decipher.com/object-drive-server/util"
)

// UpdateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to update the existing object and acm
// changing properties and permissions associated.
func (dao *DataAccessLayer) UpdateObject(object *models.ODObject) error {
	defer util.Time("UpdateObject")()
	logger := dao.GetLogger()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		logger.Error("Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	var acmCreated bool
	deadlockRetryCounter := dao.DeadlockRetryCounter
	deadlockRetryDelay := dao.DeadlockRetryDelay
	deadlockMessage := `Deadlock`
	acmCreated, err = updateObjectInTransaction(logger, tx, dao, object)
	// Deadlock trapper on acm
	for deadlockRetryCounter > 0 && err != nil && strings.Contains(err.Error(), deadlockMessage) {
		logger.Info("deadlock in UpdateObject, restarting transaction", zap.Int64("deadlockRetryCounter", deadlockRetryCounter))
		time.Sleep(time.Duration(deadlockRetryDelay) * time.Millisecond)
		// Cancel the old transaction and start a new one
		tx.Rollback()
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction", zap.String("err", err.Error()))
			return err
		}
		// Retry the create
		deadlockRetryCounter--
		acmCreated, err = updateObjectInTransaction(logger, tx, dao, object)
	}
	if err != nil {
		logger.Error("Error in UpdateObject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
		// Calculate in background and as separate transaction...
		if acmCreated {
			runasync := true
			if runasync {
				if err := insertAssociationOfACMToModifiedByIfValid(dao, *object); err != nil {
					logger.Error("Error associating the ACM on this object to the user that created it!", zap.Error(err), zap.String("ObjectID", hex.EncodeToString(object.ID)), zap.String("modifiedby", object.ModifiedBy), zap.Int64("acmID", object.ACMID))
				}

				go func() {
					done := make(chan bool)
					timeout := time.After(60 * time.Second)
					go dao.AssociateUsersToNewACM(*object, done)

					for {
						select {
						case <-timeout:
							dao.GetLogger().Warn("UpdateObject call to AssociateUsersToNewACM timed out")
							return
						case <-done:
							return
						}
					}
				}()
			} else {
				done := make(chan bool, 1)
				dao.AssociateUsersToNewACM(*object, done)
			}
		}
	}
	return err
}

func updateObjectInTransaction(logger zap.Logger, tx *sqlx.Tx, dao *DataAccessLayer, object *models.ODObject) (bool, error) {

	var acmCreated bool

	// Pre-DB Validation
	if object.ID == nil {
		return false, ErrMissingID
	}
	if object.ChangeToken == "" {
		return false, ErrMissingChangeToken
	}
	if object.ModifiedBy == "" {
		return false, ErrMissingModifiedBy
	}
	if len(object.ParentID) == 0 {
		object.ParentID = nil
	}

	// Fetch current state of object
	dbObject, err := getObjectInTransaction(tx, *object, true)
	if err != nil {
		return acmCreated, fmt.Errorf("UpdateObject Error retrieving object, %s", err.Error())
	}
	// Check if changeToken matches
	if object.ChangeToken != dbObject.ChangeToken {
		return acmCreated, util.NewLoggable("changetoken does not match expected value", nil, zap.String("changeToken", dbObject.ChangeToken))
	}
	// Check if deleted
	if dbObject.IsDeleted {
		return acmCreated, fmt.Errorf("unable to modify object if deleted. Call UndeletObject first")
	}

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := getObjectTypeByNameInTransaction(tx, object.TypeName.String, true, object.ModifiedBy)
		if err != nil {
			return acmCreated, fmt.Errorf("UpdateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// Assign a generic name if this object name is being cleared
	if len(object.Name) == 0 {
		object.Name = "Unnamed " + object.TypeName.String
	}

	// Add ownedby if a user that is not yet present.
	ownedby := object.OwnedBy.String
	if len(ownedby) > 0 {
		acmGrantee := models.NewODAcmGranteeFromResourceName(ownedby)
		if acmGrantee.UserDistinguishedName.Valid && len(acmGrantee.UserDistinguishedName.String) > 0 {
			userRequested := models.ODUser{}
			userRequested.DistinguishedName = acmGrantee.UserDistinguishedName.String
			_, err := getUserByDistinguishedNameInTransaction(tx, userRequested)
			if err != nil && err == sql.ErrNoRows {
				// Not yet in database, we need to add this user
				userRequested.DisplayName = models.ToNullString(config.GetCommonName(userRequested.DistinguishedName))
				userRequested.CreatedBy = object.CreatedBy
				userCreated := models.ODUser{}
				userCreated, err = createUserInTransaction(logger, tx, userRequested)
				object.OwnedBy = models.ToNullString("user/" + userCreated.DistinguishedName)
			}
		}
	}

	// Normalize ACM
	newACMNormalized, err := normalizedACM(object.RawAcm.String)
	if err != nil {
		return acmCreated, fmt.Errorf("Error normalizing ACM on modified object: %v (acm: %s)", err.Error(), object.RawAcm.String)
	}
	object.RawAcm.String = newACMNormalized
	if acmCreated, err = setObjectACM2ForObjectInTransaction(tx, dao, object); err != nil {
		return acmCreated, fmt.Errorf("Error assigning ACM ID for object: %v", err.Error())
	}

	// update object
	updateObjectStatement, err := tx.Preparex(`update object set 
        modifiedBy = ?
		,ownedBy = ?
        ,typeId = ?
        ,name = ?
        ,description = ?
        ,parentId = ?
        ,contentConnector = ?
        ,rawAcm = ?
        ,contentType = ?
        ,contentSize = ?
        ,contentHash = ?
        ,encryptIV = ?
        ,containsUSPersonsData = ?
        ,exemptFromFOIA = ?
        ,acmId = ?
    where id = ? and changeToken = ?`)
	if err != nil {
		return acmCreated, fmt.Errorf("UpdateObject Preparing update object statement, %s", err.Error())
	}
	defer updateObjectStatement.Close()
	// Update it
	result, err := updateObjectStatement.Exec(object.ModifiedBy, object.OwnedBy.String,
		object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String,
		object.ContentType.String, object.ContentSize, object.ContentHash,
		object.EncryptIV, object.ContainsUSPersonsData, object.ExemptFromFOIA,
		object.ACMID,
		object.ID,
		object.ChangeToken)
	if err != nil {
		return acmCreated, fmt.Errorf("UpdateObject Error executing update object statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return acmCreated, fmt.Errorf("UpdateObject Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return acmCreated, util.NewLoggable("UpdateObject did not affect any rows", nil, zap.String("id", hex.EncodeToString(object.ID)), zap.String("changetoken", object.ChangeToken))
	}

	// Compare properties on database object to properties associated with passed
	// in object
	for _, objectProperty := range object.Properties {
		addProperty := true
		// Check if existing need deleted
		for _, dbProperty := range dbObject.Properties {
			if objectProperty.Name == dbProperty.Name {
				// Delete if value is empty, differs, or classificationPM  differs
				if (len(objectProperty.Value.String) == 0) ||
					(objectProperty.Value.String != dbProperty.Value.String) ||
					(objectProperty.ClassificationPM.String != dbProperty.ClassificationPM.String) {
					// Don't readd the property if the intent is to delete
					if len(objectProperty.Value.String) == 0 {
						addProperty = false
					}
					// Deleting matching properties by name. The id and changeToken are
					// implicit from dbObject for each one that matches.
					dbProperty.ModifiedBy = object.ModifiedBy
					err = deleteObjectPropertyInTransaction(tx, dbProperty)
					if err != nil {
						return acmCreated, util.NewLoggable("error deleting property during update", err, zap.String("property.name", dbProperty.Name))
					}
					// don't break for loop here because we want to clean out all of the
					// existing properties with the same name in this case.
				} else {
					// name, value, and classificationPM are the same. dont add
					addProperty = false
				}
			}
		} // dbPropety
		if addProperty {
			// Add the newly passed in property
			var newProperty models.ODProperty
			newProperty.CreatedBy = object.ModifiedBy
			newProperty.Name = objectProperty.Name
			if objectProperty.Value.Valid {
				newProperty.Value = models.ToNullString(objectProperty.Value.String)
			}
			if objectProperty.ClassificationPM.Valid {
				newProperty.ClassificationPM = models.ToNullString(objectProperty.ClassificationPM.String)
			}
			dbProperty, err := addPropertyToObjectInTransaction(tx, *object, &newProperty)
			if err != nil {
				return acmCreated, util.NewLoggable("error saving property for object", err, zap.String("property.name", objectProperty.Name))
			}
			if dbProperty.ID == nil {
				return acmCreated, fmt.Errorf("New property does not have an ID")
			}
		}
	} //objectProperty

	// Permissions...
	// For updates, permissions are either deleted or created. It is assumed that the caller has
	// already adjusted the necessary permissions accordingly and we're simply processing the array
	// of permissions passed in without
	for permIdx, permission := range object.Permissions {
		if permission.IsDeleted && !permission.IsCreating() {
			permission.ModifiedBy = object.ModifiedBy
			deletedPermission, err := deleteObjectPermissionInTransaction(tx, permission)
			if err != nil {
				return acmCreated, fmt.Errorf("Error deleting removed permission #%d: %v", permIdx, err)
			}
			if deletedPermission.DeletedBy.String != deletedPermission.ModifiedBy {
				return acmCreated, fmt.Errorf("When deleting permission #%d, it did not get deletedBy set to modifiedBy", permIdx)
			}
		}
		if permission.IsCreating() && !permission.IsDeleted {
			permission.CreatedBy = object.ModifiedBy
			createdPermission, err := addPermissionToObjectInTransaction(logger, tx, *object, &permission)
			if err != nil {
				return acmCreated, fmt.Errorf("Error saving permission #%d {%s) when updating object:%v", permIdx, permission, err)
			}
			if createdPermission.ModifiedBy != createdPermission.CreatedBy {
				return acmCreated, fmt.Errorf("When creating permission #%d, it did not get modifiedby set to createdby", permIdx)
			}
		}
	}

	// Refetch object again with properties and permissions
	dbObject, err = getObjectInTransaction(tx, *object, true)
	if err != nil {
		return acmCreated, fmt.Errorf("UpdateObject Error retrieving object %v, %s", object, err.Error())
	}
	*object = dbObject
	return acmCreated, nil
}

func normalizedACM(i string) (string, error) {
	var normalizedInterface interface{}
	if err := json.Unmarshal([]byte(i), &normalizedInterface); err != nil {
		return i, err
	}
	normalizedBytes, err := json.Marshal(normalizedInterface)
	if err != nil {
		return i, err
	}
	return string(normalizedBytes[:]), nil
}

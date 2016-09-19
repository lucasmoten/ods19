package dao

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"

	"decipher.com/object-drive-server/util"
)

// UpdateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to update the existing object and acm
// changing properties and permissions associated.
func (dao *DataAccessLayer) UpdateObject(object *models.ODObject) error {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	err = updateObjectInTransaction(dao.GetLogger(), tx, object)
	if err != nil {
		dao.GetLogger().Error("Error in UpdateObject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func updateObjectInTransaction(logger zap.Logger, tx *sqlx.Tx, object *models.ODObject) error {

	// Pre-DB Validation
	if object.ID == nil {
		return ErrMissingID
	}
	if object.ChangeToken == "" {
		return ErrMissingChangeToken
	}
	if object.ModifiedBy == "" {
		return ErrMissingModifiedBy
	}
	if len(object.ParentID) == 0 {
		object.ParentID = nil
	}

	// Fetch current state of object
	dbObject, err := getObjectInTransaction(tx, *object, true)
	if err != nil {
		return fmt.Errorf("UpdateObject Error retrieving object, %s", err.Error())
	}
	// Check if changeToken matches
	if object.ChangeToken != dbObject.ChangeToken {
		return util.NewAppErrorInput(nil, fmt.Sprintf("Object ChangeToken does not match expected value %s", dbObject.ChangeToken))
	}
	// Check if deleted
	if dbObject.IsDeleted {
		// NOOP
		// TODO Do we need to return more information here?
		return nil
	}

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := getObjectTypeByNameInTransaction(tx, object.TypeName.String, true, object.ModifiedBy)
		if err != nil {
			return fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// Assign a generic name if this object name is being cleared
	if len(object.Name) == 0 {
		object.Name = "Unnamed " + object.TypeName.String
	}

	// Normalize ACM
	newACMNormalized, err := normalizedACM(object.RawAcm.String)
	if err != nil {
		return fmt.Errorf("Error normalizing ACM on new object: %v (acm: %s)", err.Error(), object.RawAcm.String)
	}
	object.RawAcm.String = newACMNormalized

	// update object
	updateObjectStatement, err := tx.Preparex(`update object set 
        modifiedBy = ?
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
    where id = ? and changeToken = ?`)
	if err != nil {
		return fmt.Errorf("UpdateObject Preparing update object statement, %s", err.Error())
	}
	// Update it
	result, err := updateObjectStatement.Exec(object.ModifiedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String,
		object.ContentType.String, object.ContentSize, object.ContentHash,
		object.EncryptIV, object.ContainsUSPersonsData, object.ExemptFromFOIA, object.ID,
		object.ChangeToken)
	if err != nil {
		return fmt.Errorf("UpdateObject Error executing update object statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("UpdateObject Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		jobject := mapping.MapODObjectToJSON(object)
		log.Printf("WARNING:UpdateObject did not affect any rows (Possible bad ID or changeToken)!:%s", jobject)
	}
	updateObjectStatement.Close()

	// Process ACM changes
	oldACMNormalized, err := normalizedACM(dbObject.RawAcm.String)
	if err != nil {
		return fmt.Errorf("Error normalizing ACM on database object: %s {dbObject.RawAcm: %s}", err.Error(), dbObject.RawAcm.String)
	}
	if strings.Compare(oldACMNormalized, newACMNormalized) != 0 {
		object.RawAcm.String = newACMNormalized
		err := setObjectACMForObjectInTransaction(tx, object, false)
		if err != nil { //&& err != sql.ErrNoRows {
			return fmt.Errorf("Error updating ACM for object: %s {oldacm: %s} {newacm: %s}", err.Error(), dbObject.RawAcm.String, object.RawAcm.String)
		}
	}

	// Compare properties on database object to properties associated with passed
	// in object
	for o, objectProperty := range object.Properties {
		existingProperty := false
		for _, dbProperty := range dbObject.Properties {
			if objectProperty.Name == dbProperty.Name && objectProperty.Value.Valid {
				// Updating an existing property
				existingProperty = true
				if len(objectProperty.Value.String) == 0 {
					// Deleting matching properties by name. The id and changeToken are
					// implicit from dbObject for each one that matches.
					dbProperty.ModifiedBy = object.ModifiedBy
					deleteObjectPropertyInTransaction(tx, dbProperty)
					// don't break for loop here because we want to clean out all of the
					// existing properties with the same name in this case.
				} else {
					// The name matched, but value isn't empty. Is it different?
					if (objectProperty.Value.String != dbProperty.Value.String) ||
						(objectProperty.ClassificationPM.String != dbProperty.Value.String) {
						// Existing property, but with a new value... need to update
						dbProperty.ModifiedBy = object.ModifiedBy
						dbProperty.Value.String = objectProperty.Value.String
						dbProperty.ClassificationPM.String = objectProperty.ClassificationPM.String
						updateObjectPropertyInTransaction(tx, dbProperty)
					}
					// break out of the for loop on database objects
					break
				}
			}
		} // dbPropety
		if !existingProperty {
			// Add the newly passed in property
			var newProperty models.ODProperty
			newProperty.CreatedBy = object.ModifiedBy
			newProperty.Name = objectProperty.Name
			if objectProperty.Value.Valid {
				newProperty.Value.Valid = true
				newProperty.Value.String = objectProperty.Value.String
			}
			if objectProperty.ClassificationPM.Valid {
				newProperty.ClassificationPM.Valid = true
				newProperty.ClassificationPM.String = objectProperty.ClassificationPM.String
			}
			dbProperty, err := addPropertyToObjectInTransaction(tx, *object, &newProperty)
			if err != nil {
				return fmt.Errorf("Error saving property %d (%s) when updating object:%v", o, objectProperty.Name, err)
			}
			if dbProperty.ID == nil {
				return fmt.Errorf("New property does not have an ID")
			}
		} else {
			// This existing property needs to be updated
		}
	} //objectProperty

	// Permissions...
	// For updates, we delete any readonly permissions that are not being passed in
	// and then create any new ones that aren't already present
	// This is to explicitly support changes to those permissions that have come about as a result of
	// the ACM f_share values (or lack of representing everyone)
	// Note: No other permissions outside of readonly are manipulated through this routine.
	// Permissions created explicitly are done via AddObjectPermission by way of shares
	//
	// Delete any permissions passed in that are marked as deleted on the object
	for oi := len(object.Permissions) - 1; oi >= 0; oi-- {
		permission := object.Permissions[oi]
		if permission.IsDeleted {
			permission.ModifiedBy = object.ModifiedBy
			deletedDbPermission, err := deleteObjectPermissionInTransaction(tx, permission, false)
			if err != nil {
				return fmt.Errorf("Error deleting removed permission: %v", err)
			}
			// remove from object array for processing
			object.Permissions = append(object.Permissions[:oi], object.Permissions[oi+1:]...)
			// iterate permissions on db object looking for same
			for di := len(dbObject.Permissions) - 1; di >= 0; di-- {
				if bytes.Equal(dbObject.Permissions[di].ID, deletedDbPermission.ID) {
					dbObject.Permissions[di] = deletedDbPermission
				}
			}
		}
	}
	// Delete any readonly permissions existing on the object that dont have a corresponding permission being passed
	for _, dbPermission := range dbObject.Permissions {
		if dbPermission.IsReadOnly() && !dbPermission.IsDeleted {
			dbPermissionStillValid := false
			for _, objPermission := range object.Permissions {
				if (objPermission.IsReadOnly()) && (strings.Compare(objPermission.Grantee, dbPermission.Grantee) == 0) {
					dbPermissionStillValid = true
					break
				}
			}
			if !dbPermissionStillValid {
				dbPermission.ModifiedBy = object.ModifiedBy
				_, err := deleteObjectPermissionInTransaction(tx, dbPermission, false)
				if err != nil {
					return fmt.Errorf("Error deleting obsolete permission: %v", err)
				}
			}
		}
	}
	// Add any readonly permissions being passed in that dont yet exist on the database object
	for _, objPermission := range object.Permissions {
		if objPermission.IsReadOnly() {
			objPermissionPresent := false
			for _, dbPermission := range dbObject.Permissions {
				if (dbPermission.IsReadOnly()) && (strings.Compare(objPermission.Grantee, dbPermission.Grantee) == 0) {
					objPermissionPresent = true
					break
				}
			}
			if !objPermissionPresent {
				objPermission.CreatedBy = object.ModifiedBy
				createdPermission, err := addPermissionToObjectInTransaction(logger, tx, *object, &objPermission, false, "")
				if err != nil {
					return fmt.Errorf("Error saving readonly permission {Grantee: \"%s\") when updating object:%v", objPermission.Grantee, err)
				}
				if createdPermission.ModifiedBy != createdPermission.CreatedBy {
					return fmt.Errorf("When updating object, readonly permission did not get modifiedby set to createdby")
				}
			}
		}
	}

	// Refetch object again with properties and permissions
	dbObject, err = getObjectInTransaction(tx, *object, true)
	if err != nil {
		return fmt.Errorf("UpdateObject Error retrieving object %v, %s", object, err.Error())
	}
	*object = dbObject

	return nil
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
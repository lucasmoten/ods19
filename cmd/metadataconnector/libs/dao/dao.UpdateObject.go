package dao

import (
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
)

// UpdateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to update the existing object and acm
// changing properties and permissions associated.
func (dao *DataAccessLayer) UpdateObject(object *models.ODObject, acm *models.ODACM) error {
	tx := dao.MetadataDB.MustBegin()
	err := updateObjectInTransaction(tx, object, acm)
	if err != nil {
		log.Printf("Error in UpdateObject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func updateObjectInTransaction(tx *sqlx.Tx, object *models.ODObject, acm *models.ODACM) error {

	// Pre-DB Validation
	if object.ID == nil {
		return errors.New("Object ID was not specified for object being updated")
	}
	if object.ChangeToken == "" {
		return errors.New("Object ChangeToken was not specified for object being updated")
	}
	if object.ModifiedBy == "" {
		return errors.New("Object ModifiedBy was not specified for object being updated")
	}

	// Fetch current state of object
	dbObject, err := getObjectInTransaction(tx, *object, false)
	if err != nil {
		return fmt.Errorf("UpdateObject Error retrieving object, %s", err.Error())
	}
	// Check if changeToken matches
	if object.ChangeToken != dbObject.ChangeToken {
		return fmt.Errorf("Object ChangeToken does not match expected value %s", dbObject.ChangeToken)
	}
	// Check if deleted
	if dbObject.IsDeleted {
		// NOOP
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

	// update object
	updateObjectStatement, err := tx.Preparex(
		`update object set modifiedBy = ?, typeId = ?, name = ?,
    description = ?, parentId = ?, contentConnector = ?,
    contentType = ?, contentSize = ?, contentHash = ?, encryptIV = ?
    where id = ? and changeToken = ?`)
	if err != nil {
		return fmt.Errorf("UpdateObject Preparing update object statement, %s", err.Error())
	}
	// Update it
	result, err := updateObjectStatement.Exec(object.ModifiedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.ContentType.String,
		object.ContentSize, object.ContentHash, object.EncryptIV,
		object.ID, object.ChangeToken)
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

	// TODO: Process ACM changes

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
				return fmt.Errorf("Error saving property %d (%s) when updating object", o, objectProperty.Name)
			}
			if dbProperty.ID == nil {
				return fmt.Errorf("New property does not have an ID")
			}
		} else {
			// This existing property needs to be updated
		}
	} //objectProperty

	// Refetch object again with properties and permissions
	dbObject, err = getObjectInTransaction(tx, *object, true)
	if err != nil {
		return fmt.Errorf("UpdateObject Error retrieving object %v, %s", object, err.Error())
	}
	*object = dbObject

	return nil
}

package dao

import (
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/metadata/models/acm"
)

// UpdateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to update the existing object and acm
// changing properties and permissions associated.
func (dao *DataAccessLayer) UpdateObject(object *models.ODObject) error {
	tx := dao.MetadataDB.MustBegin()
	err := updateObjectInTransaction(tx, object)
	if err != nil {
		log.Printf("Error in UpdateObject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func updateObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) error {

	// Pre-DB Validation
	if object.ID == nil {
		return errMissingID
	}
	if object.ChangeToken == "" {
		return errMissingChangeToken
	}
	if object.ModifiedBy == "" {
		return errMissingModifiedBy
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

	// Process ACM changes
	if strings.Compare(dbObject.RawAcm.String, object.RawAcm.String) != 0 {
		object.ACM.ID = dbObject.ACM.ID
		updatedACM, err := updateObjectACMForObjectInTransaction(tx, object)
		if err != nil {
			return fmt.Errorf("Error updating ACM for object: %s", err.Error())
		}
		dbObject.ACM = updatedACM
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

func updateObjectACMForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) (models.ODObjectACM, error) {

	var dbObjectACM models.ODObjectACM

	// Check if ACM is already inintialized from object
	if len(object.ACM.FlatClearance) == 0 {
		capturedACMID := object.ACM.ID
		// Clearance is required and not set. Attempt to parse and map from RawACM
		rawAcmString := object.RawAcm.String
		// Make sure its parseable
		parsedACM, err := acm.NewACMFromRawACM(rawAcmString)
		if err != nil {
			return dbObjectACM, fmt.Errorf("Cannot parse ACM: %s", err.Error())

		}
		// Map the parsed acm
		object.ACM = mapping.MapACMToODObjectACM(&parsedACM)
		object.ACM.ID = capturedACMID
	}

	// Assign based upon state of object
	object.ACM.CreatedBy = object.ModifiedBy
	object.ACM.ObjectID = object.ID

	// Set modified
	object.ACM.ModifiedBy = object.ModifiedBy

	// Update
	updateStatement, err := tx.Preparex(`update object_acm set modifiedBy = ?, 
        f_clearance = ?, f_share = ?, f_oc_org = ?, f_missions = ?, f_regions = ?, 
        f_macs = ?, f_sci_ctrls = ?, f_accms = ?, f_sar_id = ?, f_atom_energy = ?,
        f_dissem_countries = ?
        where id = ?
        `)

	if err != nil {
		return dbObjectACM, fmt.Errorf("UpdateObjectACM Preparing update statement, %s", err.Error())
	}
	result, err := updateStatement.Exec(object.ACM.ModifiedBy,
		object.ACM.FlatClearance, object.ACM.FlatShare.String, object.ACM.FlatOCOrgs.String,
		object.ACM.FlatMissions.String, object.ACM.FlatRegions.String,
		object.ACM.FlatMAC.String, object.ACM.FlatSCI.String, object.ACM.FlatACCMS.String,
		object.ACM.FlatSAR.String, object.ACM.FlatAtomEnergy.String,
		object.ACM.FlatDissemCountries.String,
		object.ACM.ID)

	if err != nil {
		return dbObjectACM, fmt.Errorf("UpdateObjectACM Error executing update statement, %s", err.Error())
	}
	err = updateStatement.Close()
	if err != nil {
		return dbObjectACM, fmt.Errorf("UpdateObjectACM Error closing update statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbObjectACM, fmt.Errorf("UpdateObjectACM Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return dbObjectACM, fmt.Errorf("UpdateObjectACM updated but no rows affected!")
	}

	// Get the newly created object_acm and return it
	// This assumes most recent object_acm created for the object that isn't deleted
	dbObjectACM, err = getObjectACMForObjectInTransaction(tx, *object, false)
	if err != nil {
		return dbObjectACM, fmt.Errorf("Error retrieving acm object: %s", err.Error())
	}

	return dbObjectACM, nil

}

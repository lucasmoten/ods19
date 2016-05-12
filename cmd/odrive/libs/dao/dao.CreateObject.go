package dao

import (
	"errors"
	"fmt"
	"log"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"github.com/jmoiron/sqlx"
)

// CreateObject ...
func (dao *DataAccessLayer) CreateObject(object *models.ODObject) (models.ODObject, error) {
	tx := dao.MetadataDB.MustBegin()
	dbObject, err := createObjectInTransaction(tx, object)
	if err != nil {
		log.Printf("Error in CreateObject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObject, err
}

func createObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) (models.ODObject, error) {

	var dbObject models.ODObject

	// Validations on object passed in.

	if len(object.TypeID) == 0 {
		object.TypeID = nil
	}
	if len(object.ParentID) == 0 {
		object.ParentID = nil
	}
	if object.CreatedBy == "" {
		return dbObject, errors.New("Cannot create object. Missing CreatedBy field.")
	}

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := getObjectTypeByNameInTransaction(tx, object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			return dbObject, fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// Assign a generic name if this object wasn't given a name
	if len(object.Name) == 0 {
		object.Name = "New " + object.TypeName.String
	}

	// Insert object
	addObjectStatement, err := tx.Preparex(`insert object set 
        createdBy = ?
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
        ,isUSPersonsData = ?
        ,isFOIAExempt = ?        
    `)
	if err != nil {
		return dbObject, fmt.Errorf("CreateObject Preparing add object statement, %s", err.Error())
	}
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String,
		object.ContentType.String, object.ContentSize.Int64, object.ContentHash,
		object.EncryptIV, object.IsUSPersonsData, object.IsFOIAExempt)
	if err != nil {
		return dbObject, fmt.Errorf("CreateObject Error executing add object statement, %s", err.Error())
	}
	err = addObjectStatement.Close()
	if err != nil {
		return dbObject, fmt.Errorf("CreateObject Error closing addObjectStatement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbObject, fmt.Errorf("CreateObject Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return dbObject, fmt.Errorf("CreateObject object inserted but no rows affected!")
	}

	// Get the ID of the newly created object and assign to returned object.
	// This assumes most recent created by the user of the type and name.
	getObjectStatement := `
    select 
        o.id    
        ,o.createdDate
        ,o.createdBy
        ,o.modifiedDate
        ,o.modifiedBy
        ,o.isDeleted
        ,o.deletedDate
        ,o.deletedBy
        ,o.isAncestorDeleted
        ,o.isExpunged
        ,o.expungedDate
        ,o.expungedBy
        ,o.changeCount
        ,o.changeToken
        ,o.ownedBy
        ,o.typeId
        ,o.name
        ,o.description
        ,o.parentId
        ,o.contentConnector
        ,o.rawAcm
        ,o.contentType
        ,o.contentSize
        ,o.contentHash
        ,o.encryptIV
        ,o.ownedByNew
        ,o.isPDFAvailable
        ,o.isStreamStored
        ,o.isUSPersonsData
        ,o.isFOIAExempt
        ,ot.name typeName   
    from object o 
        inner join object_type ot on o.typeId = ot.id 
    where 
        o.createdby = ? 
        and o.typeId = ? 
        and o.name = ? 
        and o.isdeleted = 0 
    order by o.createddate desc limit 1`
	err = tx.Get(&dbObject, getObjectStatement, object.CreatedBy, object.TypeID, object.Name)
	if err != nil {
		return dbObject, fmt.Errorf("CreateObject Error retrieving object, %s", err.Error())
	}

	// Add properties of object.Properties []models.ODObjectPropertyEx
	for i, property := range object.Properties {
		if property.Name != "" {
			var objectProperty models.ODProperty
			objectProperty.CreatedBy = dbObject.CreatedBy
			objectProperty.Name = property.Name
			if property.Value.Valid {
				objectProperty.Value.String = property.Value.String
				objectProperty.Value.Valid = true
			}
			if property.ClassificationPM.Valid {
				objectProperty.ClassificationPM.String = property.ClassificationPM.String
				objectProperty.ClassificationPM.Valid = true
			}
			dbProperty, err := addPropertyToObjectInTransaction(tx, dbObject, &objectProperty)
			if err != nil {
				return dbObject, fmt.Errorf("Error saving property %d (%s) when creating object", i, property.Name)
			}
			if dbProperty.ID == nil {
				return dbObject, fmt.Errorf("New property does not have an ID")
			}
		}
	}

	// Add permissions
	for i, permission := range object.Permissions {
		if permission.Grantee != "" {
			permission.CreatedBy = dbObject.CreatedBy
			dbPermission, err := addPermissionToObjectInTransaction(tx, dbObject, &permission, false, "")
			if err != nil {
				return dbObject, fmt.Errorf("Error saving permission # %d {Grantee: \"%s\") when creating object:%v", i, permission.Grantee, err)
			}
			if dbPermission.ModifiedBy != permission.CreatedBy {
				return dbObject, fmt.Errorf("When creating object, permission did not get modifiedby set to createdby")
			}

		}
	}

	// Initialize acm
	object.ACM.CreatedBy = dbObject.CreatedBy
	object.ACM.ID = dbObject.ID
	dbObject.ACM = object.ACM
	createdACM, err := createObjectACMForObjectInTransaction(tx, &dbObject)
	if err != nil {
		return dbObject, fmt.Errorf("Error saving ACM to object: %s", err.Error())
	}
	dbObject.ACM = createdACM

	return dbObject, nil
}

func createObjectACMForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) (models.ODObjectACM, error) {

	var dbObjectACM models.ODObjectACM

	// Check if ACM is already inintialized from object
	if len(object.ACM.FlatClearance) == 0 {
		// Clearance is required and not set. Attempt to parse and map from RawACM
		rawAcmString := object.RawAcm.String
		// Make sure its parseable
		parsedACM, err := acm.NewACMFromRawACM(rawAcmString)
		if err != nil {
			return dbObjectACM, fmt.Errorf("Cannot parse ACM: %s", err.Error())

		}
		// Map the parsed acm
		object.ACM = mapping.MapACMToODObjectACM(&parsedACM)
	}

	// Assign based upon state of object
	object.ACM.CreatedBy = object.ModifiedBy
	object.ACM.ObjectID = object.ID

	// Insert object
	addStatement, err := tx.Preparex(`insert object_acm set createdBy = ?, objectId = ?, acmId = null,
        f_clearance = ?, f_share = ?, f_oc_org = ?, f_missions = ?, f_regions = ?, 
        f_macs = ?, f_sci_ctrls = ?, f_accms = ?, f_sar_id = ?, f_atom_energy = ?,
        f_dissem_countries = ?`)
	if err != nil {
		return dbObjectACM, fmt.Errorf("CreateObjectACM Preparing add statement, %s", err.Error())
	}
	result, err := addStatement.Exec(object.ACM.CreatedBy, object.ACM.ObjectID,
		object.ACM.FlatClearance, object.ACM.FlatShare.String, object.ACM.FlatOCOrgs.String,
		object.ACM.FlatMissions.String, object.ACM.FlatRegions.String,
		object.ACM.FlatMAC.String, object.ACM.FlatSCI.String, object.ACM.FlatACCMS.String,
		object.ACM.FlatSAR.String, object.ACM.FlatAtomEnergy.String,
		object.ACM.FlatDissemCountries.String)

	if err != nil {
		return dbObjectACM, fmt.Errorf("CreateObjectACM Error executing add statement, %s", err.Error())
	}
	err = addStatement.Close()
	if err != nil {
		return dbObjectACM, fmt.Errorf("CreateObjectACM Error closing add statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbObjectACM, fmt.Errorf("CreateObjectACM Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return dbObjectACM, fmt.Errorf("CreateObjectACM inserted but no rows affected!")
	}

	// Get the newly created object_acm and return it
	// This assumes most recent object_acm created for the object that isn't deleted
	dbObjectACM, err = getObjectACMForObjectInTransaction(tx, *object, false)
	if err != nil {
		return dbObjectACM, fmt.Errorf("Error retrieving acm object: %s", err.Error())
	}

	return dbObjectACM, nil

}

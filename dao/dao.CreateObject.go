package dao

import (
	"errors"
	"fmt"

	"decipher.com/object-drive-server/utils"
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// CreateObject ...
func (dao *DataAccessLayer) CreateObject(object *models.ODObject) (models.ODObject, error) {
	logger := dao.GetLogger()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.String("err", err.Error()))
		return models.ODObject{}, err
	}
	dbObject, err := createObjectInTransaction(logger, tx, object)
	if err != nil {
		logger.Error("error in CreateObject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	obj, err := dao.GetObject(dbObject, true)
	if err != nil {
		logger.Error("error in CreateObject subsequent GetObject call]")
		return models.ODObject{}, err
	}
	return obj, err
}

func createObjectInTransaction(logger zap.Logger, tx *sqlx.Tx, object *models.ODObject) (models.ODObject, error) {

	var dbObject models.ODObject

	if len(object.TypeID) == 0 {
		object.TypeID = nil
	}
	if len(object.ParentID) == 0 {
		object.ParentID = nil
	}
	if object.CreatedBy == "" {
		return dbObject, errors.New("Cannot create object. Missing CreatedBy field.")
	}

	if object.TypeID == nil {
		objectType, err := getObjectTypeByNameInTransaction(tx, object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			return dbObject, fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	if len(object.Name) == 0 {
		object.Name = "New " + object.TypeName.String
	}

	// Assign a random content connector value if this object doesnt have one
	if len(object.ContentConnector.String) == 0 {
		object.ContentConnector = models.ToNullString(utils.CreateRandomName())
	}

	// Normalize ACM
	newACMNormalized, err := normalizedACM(object.RawAcm.String)
	if err != nil {
		return dbObject, fmt.Errorf("Error normalizing ACM on new object: %v (acm: %s)", err.Error(), object.RawAcm.String)
	}
	object.RawAcm.String = newACMNormalized

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
        ,containsUSPersonsData = ?
        ,exemptFromFOIA = ?        
    `)
	if err != nil {
		return dbObject, fmt.Errorf("CreateObject Preparing add object statement, %s", err.Error())
	}
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String,
		object.ContentType.String, object.ContentSize.Int64, object.ContentHash,
		object.EncryptIV, object.ContainsUSPersonsData, object.ExemptFromFOIA)
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
		return dbObject, fmt.Errorf("CreateObject object inserted but no rows affected")
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
        ,o.containsUSPersonsData
        ,o.exemptFromFOIA
        ,ot.name typeName   
    from object o 
        inner join object_type ot on o.typeId = ot.id 
    where 
        o.createdby = ? 
        and o.typeId = ? 
        and o.name = ? 
        and o.contentConnector = ?
        and o.isdeleted = 0 
    order by o.createddate desc limit 1`
	err = tx.Get(&dbObject, getObjectStatement, object.CreatedBy, object.TypeID, object.Name, object.ContentConnector)
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
		if !permission.IsDeleted && permission.Grantee != "" {
			permission.CreatedBy = dbObject.CreatedBy
			dbPermission, err := addPermissionToObjectInTransaction(logger, tx, dbObject, &permission, false, "")
			if err != nil {
				return dbObject, fmt.Errorf("Error saving permission # %d {Grantee: \"%s\") when creating object:%v", i, permission.Grantee, err)
			}
			if dbPermission.ModifiedBy != permission.CreatedBy {
				return dbObject, fmt.Errorf("When creating object, permission did not get modifiedby set to createdby")
			}
			object.Permissions[i] = dbPermission
		}
	}

	// Initialize acm
	err = setObjectACMForObjectInTransaction(tx, &dbObject, true)
	if err != nil {
		return dbObject, fmt.Errorf("Error saving ACM to object: %s", err.Error())
	}

	return dbObject, nil
}

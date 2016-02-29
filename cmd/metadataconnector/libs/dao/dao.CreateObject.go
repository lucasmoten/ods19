package dao

import (
	"fmt"
	"log"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// CreateObject ...
func (dao *DataAccessLayer) CreateObject(object *models.ODObject, acm *models.ODACM) (models.ODObject, error) {
	tx := dao.MetadataDB.MustBegin()
	dbObject, err := createObjectInTransaction(tx, object, acm)
	if err != nil {
		log.Printf("Error in CreateObject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObject, err
}

func createObjectInTransaction(tx *sqlx.Tx, object *models.ODObject, acm *models.ODACM) (models.ODObject, error) {

	var dbObject models.ODObject

	if len(object.TypeID) == 0 {
		//log.Println("Converting object TypeID from zero length byte slice to nil.")
		object.TypeID = nil
	}

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := getObjectTypeByNameInTransaction(tx, object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			return dbObject, fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// Insert object
	addObjectStatement, err := tx.Preparex(`insert object set createdBy = ?, typeId = ?, name = ?, description = ?, parentId = ?, contentConnector = ?, rawAcm = ?, contentType = ?, contentSize = ?, contentHash = ?, encryptIV = ?`)
	if err != nil {
		return dbObject, fmt.Errorf("CreateObject Preparing add object statement, %s", err.Error())
	}
	if object.ContentSize.Int64 < 0 {
		return dbObject, fmt.Errorf("Impossible file size:%d", object.ContentSize.Int64)
	}
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String, object.ContentType.String,
		object.ContentSize.Int64, object.ContentHash, object.EncryptIV)
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

	// Get the ID of the newly created object and assign to passed in object
	// This assumes most recent created by the user of the type and name
	getObjectStatement := `select o.*, ot.name typeName from object o inner join object_type ot on o.typeId = ot.id where o.createdby = ? and o.typeId = ? and o.name = ? and o.isdeleted = 0 order by o.createddate desc limit 1`
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
			dbPermission, err := addPermissionToObjectInTransaction(tx, dbObject, &permission)
			if err != nil {
				crud := []string{"C", "R", "U", "D"}
				if !permission.AllowCreate {
					crud[0] = "-"
				}
				if !permission.AllowRead {
					crud[1] = "-"
				}
				if !permission.AllowUpdate {
					crud[2] = "-"
				}
				if !permission.AllowDelete {
					crud[3] = "-"
				}
				return dbObject, fmt.Errorf("Error saving permission # %d {Grantee: \"%s\", Permission: \"%s\") when creating object:%v", i, permission.Grantee, crud, err)
			}
			if dbPermission.ModifiedBy != permission.CreatedBy {
				return dbObject, fmt.Errorf("When creating object, permision did not get modifiedby set to createdby")
			}

		}
	}

	// insert acm

	return dbObject, nil
}

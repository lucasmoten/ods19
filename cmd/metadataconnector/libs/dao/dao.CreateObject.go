package dao

import (
	"fmt"

	"decipher.com/oduploader/metadata/models"
)

func (dao *DataAccessLayer) CreateObject(object *models.ODObject, acm *models.ODACM) error {

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := dao.GetObjectTypeByName(object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			return fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// insert object
	addObjectStatement, err := dao.MetadataDB.Prepare(`insert object set createdBy = ?, typeId = ?, name = ?, description = ?, parentId = ?, contentConnector = ?, rawAcm = ?, contentType = ?, contentSize = ?, contentHash = ?, encryptIV = ?`)
	if err != nil {
		return fmt.Errorf("CreateObject Preparing add object statement, %s", err.Error())
	}
	// Add it
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String, object.ContentType.String,
		object.ContentSize.Int64, object.ContentHash, object.EncryptIV)
	if err != nil {
		return fmt.Errorf("CreateObject Error executing add object statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("CreateObject Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return fmt.Errorf("CreateObject object inserted but no rows affected!")
	}
	// Get the ID of the newly created object and assign to passed in object
	// This assumes most recent created by the user of the type and name
	getObjectStatement := `select o.*, ot.name typeName from object o inner join object_type ot on o.typeId = ot.id where o.createdby = ? and o.typeId = ? and o.name = ? and o.isdeleted = 0 order by o.createddate desc limit 1`
	err = dao.MetadataDB.Get(object, getObjectStatement, object.CreatedBy, object.TypeID, object.Name)
	if err != nil {
		return fmt.Errorf("CreateObject Error retrieving object, %s", err.Error())
	}
	// Add properties of object.Properties []models.ODObjectPropertyEx
	for i, property := range object.Properties {
		if property.Name != "" {
			var objectProperty models.ODProperty
			objectProperty.CreatedBy = object.CreatedBy
			objectProperty.Name = property.Name
			if property.Value.Valid {
				objectProperty.Value.String = property.Value.String
				objectProperty.Value.Valid = true
			}
			if property.ClassificationPM.Valid {
				objectProperty.ClassificationPM.String = property.ClassificationPM.String
				objectProperty.ClassificationPM.Valid = true
			}
			err := dao.AddPropertyToObject(object.CreatedBy, object, &objectProperty)
			if err != nil {
				return fmt.Errorf("Error saving property %d (%s) when creating object", i, property.Name)
			}
		}
	}

	// Add permissions
	for i, permission := range object.Permissions {
		if permission.Grantee != "" {
			err := dao.AddPermissionToObject(object.CreatedBy, object, &permission)
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
				return fmt.Errorf("Error saving permission # %d {Grantee: \"%s\", Permission: \"%s\") when creating object:%v", i, permission.Grantee, crud, err)
			}

		}
	}

	// insert acm

	return nil
}

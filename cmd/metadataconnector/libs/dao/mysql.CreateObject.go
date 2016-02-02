package dao

import (
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// CreateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to insert the object, insert the acm
// configuration, associate the two together. Identifiers are captured and
// assigned to the relevant objects
func CreateObject(db *sqlx.DB, object *models.ODObject, acm *models.ODACM) error {

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := GetObjectTypeByName(db, object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			return fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// insert object
	addObjectStatement, err := db.Prepare(`insert object set createdBy = ?, typeId = ?, name = ?, description = ?, parentId = ?, contentConnector = ?, contentType = ?, contentSize = ?, contentHash = ?, encryptIV = ?`)
	if err != nil {
		return fmt.Errorf("CreateObject Preparing add object statement, %s", err.Error())
	}
	// Add it
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.ContentType.String,
		object.ContentSize, object.ContentHash.String, object.EncryptIV)
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
	getObjectStatement := `select * from object where createdby = ? and typeId = ? and name = ? and isdeleted = 0 order by createddate desc limit 1`
	err = db.Get(object, getObjectStatement, object.CreatedBy, object.TypeID, object.Name)
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
			err := AddPropertyToObject(db, object.CreatedBy, object, &objectProperty)
			if err != nil {
				return fmt.Errorf("Error saving property %d (%s) when creating object", i, property.Name)
			}
		}
	}

	// Add permissions
	for i, permission := range object.Permissions {
		if permission.Grantee != "" {
			err := AddPermissionToObject(db, object.CreatedBy, object, &permission)
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

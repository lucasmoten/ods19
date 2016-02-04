package dao

import (
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// UpdateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to update the existing object and acm
// changing properties and permissions associated.
func UpdateObject(db *sqlx.DB, object *models.ODObject, acm *models.ODACM) error {

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := GetObjectTypeByName(db, object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			return fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// update object
	updateObjectStatement, err := db.Prepare(`update object set modifiedBy = ?, typeId = ?, name = ?, description = ?, parentId = ?, contentConnector = ?, contentType = ?, contentSize = ?, contentHash = ?, encryptIV = ? where id = ? and changeToken = ?`)
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
		return fmt.Errorf("UpdateObject did not affect any rows (Possible bad ID or changeToken)!")
	}

	// Retrieve current state of object from database to reflect alterations to..
	// ModifiedDate, ChangeToken, ChangeCount
	dbObject, err := GetObject(db, object, true)
	if err != nil {
		return fmt.Errorf("UpdateObject Error retrieving object, %s", err.Error())
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
					// Deleting properties of this name
					dbProperty.ModifiedBy = object.ModifiedBy
					DeleteObjectProperty(db, &dbProperty)
				} else {
					// The property exists, and may or may not have changed. For now, this
					// is going to delete the properties and re-add them below.  Ideally,
					// we need to compare all the values passed in, compared to all the
					// values present in the db properties of the same name to see which
					// are new values, and which are no longer present to perform isolated
					// delete and adds.
					// TODO: Change the logic here per the above to do less db calls and
					// keep accurate change history.
					dbProperty.ModifiedBy = object.ModifiedBy
					DeleteObjectProperty(db, &dbProperty)
					// causes it to be readded later.
					existingProperty = false
				}
			}
		}
		if !existingProperty {
			// Add the passed in property
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
			err := AddPropertyToObject(db, object.CreatedBy, object, &newProperty)
			if err != nil {
				return fmt.Errorf("Error saving property %d (%s) when updating object", o, objectProperty.Name)
			}
		}
	}

	// Permissions
	// Iterate permissions passed with the object
	for o, objectPermission := range object.Permissions {
		existingPermission := false
		// And iterate the permissions currently on the object in the database
		for _, dbPermission := range dbObject.Permissions {
			// If its the same user... (and hencec forcing collapse to only one per grantee)
			if objectPermission.Grantee == dbPermission.Grantee {
				existingPermission = true
				// See if the permission is the same...
				if objectPermission.AllowCreate != dbPermission.AllowCreate ||
					objectPermission.AllowRead != dbPermission.AllowRead ||
					objectPermission.AllowUpdate != dbPermission.AllowUpdate ||
					objectPermission.AllowDelete != dbPermission.AllowDelete {
					// The permission is different, we need to do an update on the record
					dbPermission.ModifiedBy = object.ModifiedBy
					dbPermission.AllowCreate = objectPermission.AllowCreate
					dbPermission.AllowRead = objectPermission.AllowRead
					dbPermission.AllowUpdate = objectPermission.AllowUpdate
					dbPermission.AllowDelete = objectPermission.AllowUpdate
					// Dont update EncryptKey here. That should only be updated when
					// UpdateContentStream is called
					err := UpdatePermission(db, &dbPermission)
					if err != nil {
						return fmt.Errorf("Error updating permission %d (%s) when updating object", o, objectPermission.Grantee)
					}
				}
			}
		}
		if !existingPermission {
			// No existing permission. Need to add it
			err := AddPermissionToObject(db, object.ModifiedBy, object, &objectPermission)
			if err != nil {
				crud := []string{"C", "R", "U", "D"}
				if !objectPermission.AllowCreate {
					crud[0] = "-"
				}
				if !objectPermission.AllowRead {
					crud[1] = "-"
				}
				if !objectPermission.AllowUpdate {
					crud[2] = "-"
				}
				if !objectPermission.AllowDelete {
					crud[3] = "-"
				}
				return fmt.Errorf("Error saving permission # %d {Grantee: \"%s\", Permission: \"%s\") when creating object", o, objectPermission.Grantee, crud)
			}
		}
	}

	// Refetch object again with properties and permissions
	object, err = GetObject(db, object, true)
	if err != nil {
		return fmt.Errorf("UpdateObject Error retrieving object, %s", err.Error())
	}

	return nil
}

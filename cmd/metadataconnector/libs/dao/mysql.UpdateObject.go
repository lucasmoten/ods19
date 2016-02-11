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
					DeleteObjectProperty(db, &dbProperty)
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
						UpdateObjectProperty(db, &dbProperty)
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
			err := AddPropertyToObject(db, object.CreatedBy, object, &newProperty)
			if err != nil {
				return fmt.Errorf("Error saving property %d (%s) when updating object", o, objectProperty.Name)
			}
		} else {
			// This existing property needs to be updated
		}
	} //objectProperty

	// Permissions
	// Iterate permissions passed with the object
	for o, objectPermission := range object.Permissions {
		existingPermission := false
		// And iterate the permissions currently on the object in the database
		for _, dbPermission := range dbObject.Permissions {
			// If its the same user... (and hencec forcing collapse to only one per grantee)
			if objectPermission.Grantee == dbPermission.Grantee {
				existingPermission = true
				// if the permission is not the same... we need to update it
				if objectPermission.AllowCreate != dbPermission.AllowCreate ||
					objectPermission.AllowRead != dbPermission.AllowRead ||
					objectPermission.AllowUpdate != dbPermission.AllowUpdate ||
					objectPermission.AllowDelete != dbPermission.AllowDelete {
					// The permission is different, we need to do an update on the record
					dbPermission.ModifiedBy = object.ModifiedBy
					// TODO: Should EncrypKey be updated? Seems like it would need to be
					// assigned if the user didn't have AllowRead beforehand
					if !dbPermission.AllowRead && objectPermission.AllowRead {
						// TODO: Need to assign new EncryptKey value here and possibly do
						// something to the stream? Check with Rob Fielding
					}
					dbPermission.AllowCreate = objectPermission.AllowCreate
					dbPermission.AllowRead = objectPermission.AllowRead
					dbPermission.AllowUpdate = objectPermission.AllowUpdate
					dbPermission.AllowDelete = objectPermission.AllowUpdate
					err := UpdatePermission(db, &dbPermission)
					if err != nil {
						return fmt.Errorf("Error updating permission %d (%s) when updating object", o, objectPermission.Grantee)
					}
				}
			}
		}
		if !existingPermission {
			// No existing permission. Need to add it
			// TODO: Since this is a new permission, we need to establish the
			// encryptKey for this grantee.
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

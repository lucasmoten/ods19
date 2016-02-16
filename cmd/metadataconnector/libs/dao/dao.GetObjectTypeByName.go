package dao

import (
	"database/sql"
	"fmt"

	"decipher.com/oduploader/metadata/models"
)

// GetObjectTypeByName looks up an object type by its name, and if it doesn't
// exist, optionally calls CreateObjectType to add it.
func (dao *DataAccessLayer) GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {
	var objectType models.ODObjectType
	// Get the ID of the newly created object and assign to passed in object
	getObjectTypeStatement := `select * from object_type where name = ? order by isdeleted asc, createddate desc limit 1`
	err := dao.MetadataDB.Get(&objectType, getObjectTypeStatement, typeName)
	if err != nil {
		if err == sql.ErrNoRows {
			if addIfMissing {
				objectType.Name = typeName
				objectType.CreatedBy = createdBy
				err = dao.CreateObjectType(&objectType)
			}
		} else {
			return objectType, fmt.Errorf("GetObjectTypeByName error, %s", err.Error())
		}
	}
	if objectType.IsDeleted && addIfMissing {
		objectType.Name = typeName
		objectType.CreatedBy = createdBy
		err = dao.CreateObjectType(&objectType)
	}

	return objectType, err
}

package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetChildObjectsWithProperties retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified parent
// object
func GetChildObjectsWithProperties(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error) {
	response, err := GetChildObjects(db, orderByClause, pageNumber, pageSize, object)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := GetPropertiesForObject(db, response.Objects[i].ID)
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

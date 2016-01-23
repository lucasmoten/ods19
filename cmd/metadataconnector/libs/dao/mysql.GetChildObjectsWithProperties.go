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
	for _, responseObject := range response.Objects {
		properties, err := GetPropertiesForObject(db, responseObject.ID)
		if err != nil {
			return response, err
		}
		responseObject.Properties = properties
	}
	return response, err
}

package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetChildObjectsWithPropertiesByOwner retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified object by
// parentID and are owned by the specified user or group.
func GetChildObjectsWithPropertiesByOwner(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, object *models.ODObject, owner string) (models.ODObjectResultset, error) {
	response, err := GetChildObjectsByOwner(db, orderByClause, pageNumber, pageSize, object, owner)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for _, responseObject := range response.Objects {
		properties, err := GetPropertiesForObject(db, responseObject.ID)
		if err != nil {
			print(err.Error())
			return response, err
		}
		responseObject.Properties = *properties
	}
	return response, err
}

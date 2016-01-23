package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetRootObjectsWithPropertiesByOwner retrieves a list of Objects and their
// Properties in Object Drive that are not nested beneath any other objects
// natively (natural parentId is null) and are owned by the specified user or
// group.
func GetRootObjectsWithPropertiesByOwner(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error) {
	response, err := GetRootObjectsByOwner(db, orderByClause, pageNumber, pageSize, owner)
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

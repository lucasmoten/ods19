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

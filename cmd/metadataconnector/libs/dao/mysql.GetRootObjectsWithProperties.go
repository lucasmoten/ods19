package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetRootObjectsWithProperties retrieves a list of Objects and their Properties
// in Object Drive that are not nested beneath any other objects natively
// (natural parentId is null)
func GetRootObjectsWithProperties(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	response, err := GetRootObjects(db, orderByClause, pageNumber, pageSize)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for _, object := range response.Objects {
		properties, err := GetPropertiesForObject(db, object.ID)
		if err != nil {
			print(err.Error())
			return response, err
		}
		object.Properties = *properties
	}
	return response, err
}

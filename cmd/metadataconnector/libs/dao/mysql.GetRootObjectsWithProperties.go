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
	for i := 0; i < len(response.Objects); i++ {
		properties, err := GetPropertiesForObject(db, &response.Objects[i])
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

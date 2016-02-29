package dao

import (
	"log"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetChildObjectsWithPropertiesByUser retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified object by
// parentID and are owned by the specified user or group.
func (dao *DataAccessLayer) GetChildObjectsWithPropertiesByUser(
	orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getChildObjectsWithPropertiesByUserInTransaction(tx, orderByClause, pageNumber, pageSize, object, user)
	if err != nil {
		log.Printf("Error in GetChildObjectsWithPropertiesByUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsWithPropertiesByUserInTransaction(tx *sqlx.Tx, orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {

	response, err := getChildObjectsByUserInTransaction(tx, orderByClause, pageNumber, pageSize, object, user)
	if err != nil {
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := getPropertiesForObjectInTransaction(tx, response.Objects[i])
		if err != nil {
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

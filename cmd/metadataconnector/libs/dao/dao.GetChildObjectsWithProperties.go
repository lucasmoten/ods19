package dao

import (
	"log"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetChildObjectsWithProperties retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the parent object.
func (dao *DataAccessLayer) GetChildObjectsWithProperties(
	orderByClause string, pageNumber int, pageSize int, object models.ODObject) (models.ODObjectResultset, error) {

	tx := dao.MetadataDB.MustBegin()
	response, err := getChildObjectsWithPropertiesInTransaction(tx, orderByClause, pageNumber, pageSize, object)
	if err != nil {
		log.Printf("Error in GetChildObjectsWithProperties: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsWithPropertiesInTransaction(tx *sqlx.Tx, orderByClause string, pageNumber int, pageSize int, object models.ODObject) (models.ODObjectResultset, error) {

	response, err := getChildObjectsInTransaction(tx, orderByClause, pageNumber, pageSize, object)
	if err != nil {
		print(err.Error())
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

package dao

import (
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"github.com/jmoiron/sqlx"
)

// GetRootObjectsWithProperties retrieves a list of Objects and their Properties
// in Object Drive that are not nested beneath any other objects natively
// (natural parentId is null)
func (dao *DataAccessLayer) GetRootObjectsWithProperties(pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getRootObjectsWithPropertiesInTransaction(tx, pagingRequest)
	if err != nil {
		log.Printf("Error in GetRootObjectsWithProperties: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsWithPropertiesInTransaction(tx *sqlx.Tx, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {

	response, err := getRootObjectsInTransaction(tx, pagingRequest)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := getPropertiesForObjectInTransaction(tx, response.Objects[i])
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

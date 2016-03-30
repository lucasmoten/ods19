package dao

import (
	"log"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"github.com/jmoiron/sqlx"
)

// GetRootObjectsWithPropertiesByUser retrieves a list of Objects and their
// Properties in Object Drive that are not nested beneath any other objects
// natively (natural parentId is null) and are owned by the specified user or
// group.
func (dao *DataAccessLayer) GetRootObjectsWithPropertiesByUser(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getRootObjectsWithPropertiesByUserInTransaction(tx, user, pagingRequest)
	if err != nil {
		log.Printf("Error in GetRootObjectsWithPropertiesByUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsWithPropertiesByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {

	response, err := getRootObjectsByUserInTransaction(tx, user, pagingRequest)
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

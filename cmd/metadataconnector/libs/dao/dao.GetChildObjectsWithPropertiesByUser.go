package dao

import (
	"log"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"github.com/jmoiron/sqlx"
)

// GetChildObjectsWithPropertiesByUser retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified object by
// parentID and are owned by the specified user or group.
func (dao *DataAccessLayer) GetChildObjectsWithPropertiesByUser(
	user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getChildObjectsWithPropertiesByUserInTransaction(tx, user, pagingRequest, object)
	if err != nil {
		log.Printf("Error in GetChildObjectsWithPropertiesByUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsWithPropertiesByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {

	response, err := getChildObjectsByUserInTransaction(tx, user, pagingRequest, object)
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

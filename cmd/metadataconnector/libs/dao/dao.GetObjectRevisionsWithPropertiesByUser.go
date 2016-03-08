package dao

import (
	"log"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectRevisionsWithPropertiesByUser retrieves a list of revisions for an
// object and the properties that were active at the point of that revision
func (dao *DataAccessLayer) GetObjectRevisionsWithPropertiesByUser(
	orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getObjectRevisionsWithPropertiesByUserInTransaction(tx, orderByClause, pageNumber, pageSize, object, user)
	if err != nil {
		log.Printf("Error in GetObjectRevisionsWithPropertiesByUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectRevisionsWithPropertiesByUserInTransaction(tx *sqlx.Tx, orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {

	response, err := getObjectRevisionsByUserInTransaction(tx, orderByClause, pageNumber, pageSize, object, user)
	if err != nil {
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := getPropertiesForObjectRevisionInTransaction(tx, response.Objects[i])
		if err != nil {
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

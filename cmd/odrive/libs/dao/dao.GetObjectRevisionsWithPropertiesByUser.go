package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetObjectRevisionsWithPropertiesByUser retrieves a list of revisions for an
// object and the properties that were active at the point of that revision
func (dao *DataAccessLayer) GetObjectRevisionsWithPropertiesByUser(
	user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectRevisionsWithPropertiesByUserInTransaction(tx, user, pagingRequest, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectRevisionsWithPropertiesByUser", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectRevisionsWithPropertiesByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {

	response, err := getObjectRevisionsByUserInTransaction(tx, user, pagingRequest, object)
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

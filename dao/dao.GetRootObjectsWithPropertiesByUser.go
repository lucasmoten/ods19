package dao

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetRootObjectsWithPropertiesByUser retrieves a list of Objects and their
// Properties in Object Drive that are not nested beneath any other objects
// natively (natural parentId is null) and are owned by the specified user or
// group.
func (dao *DataAccessLayer) GetRootObjectsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetRootObjectsWithPropertiesByUser")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsWithPropertiesByUserInTransaction(tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjectsWithPropertiesByUser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsWithPropertiesByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response, err := getRootObjectsByUserInTransaction(tx, user, pagingRequest)
	if err != nil {
		return response, err
	}
	return response, err
}

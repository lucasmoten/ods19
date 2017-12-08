package dao

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetRootObjectsWithProperties retrieves a list of Objects and their Properties
// in Object Drive that are not nested beneath any other objects natively
// (natural parentId is null)
func (dao *DataAccessLayer) GetRootObjectsWithProperties(pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetRootObjectsWithProperties")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsWithPropertiesInTransaction(tx, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjectsWithProperties", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsWithPropertiesInTransaction(tx *sqlx.Tx, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response, err := getRootObjectsInTransaction(tx, pagingRequest)
	if err != nil {
		return response, err
	}
	return response, err
}

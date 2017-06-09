package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetChildObjectsWithProperties retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the parent object.
func (dao *DataAccessLayer) GetChildObjectsWithProperties(
	pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	defer util.Time("GetChildObjectsWithProperties")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getChildObjectsWithPropertiesInTransaction(tx, pagingRequest, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetChildObjectsWithProperties", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsWithPropertiesInTransaction(tx *sqlx.Tx, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	return getChildObjectsInTransaction(tx, pagingRequest, object, true)
}

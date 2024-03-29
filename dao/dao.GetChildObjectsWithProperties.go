package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"go.uber.org/zap"
)

// GetChildObjectsWithProperties retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the parent object.
func (dao *DataAccessLayer) GetChildObjectsWithProperties(
	pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	defer util.Time("GetChildObjectsWithProperties")()
	loadPermissions := true
	loadProperties := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getChildObjectsInTransaction(dao, tx, pagingRequest, object, loadPermissions, loadProperties)
	if err != nil {
		dao.GetLogger().Error("error in getchildobjectswithproperties", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

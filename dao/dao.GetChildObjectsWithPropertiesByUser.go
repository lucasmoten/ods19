package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetChildObjectsWithPropertiesByUser retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified object by
// parentID and are owned by the specified user or group.
func (dao *DataAccessLayer) GetChildObjectsWithPropertiesByUser(
	user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	defer util.Time("GetChildObjectsWithPropertiesByUser")()
	loadPermissions := true
	loadProperties := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getChildObjectsWithPropertiesByUserInTransaction(tx, user, pagingRequest, object, loadPermissions, loadProperties)
	if err != nil {
		dao.GetLogger().Error("error in getchildobjectswithpropertiesbyuser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsWithPropertiesByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, object models.ODObject, loadPermissions bool, loadProperties bool) (models.ODObjectResultset, error) {

	response, err := getChildObjectsByUserInTransaction(tx, user, pagingRequest, object, loadPermissions, loadProperties)
	if err != nil {
		return response, err
	}
	return response, err
}

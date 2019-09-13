package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"go.uber.org/zap"
)

// GetRootObjectsWithPropertiesByGroup retrieves a list of Objects and their
// Properties in Object Drive that are not nested beneath any other objects
// natively (natural parentId is null) and are owned by the specified group
func (dao *DataAccessLayer) GetRootObjectsWithPropertiesByGroup(groupGranteeName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetRootObjectsWithPropertiesByGroup")()
	loadPermissions := true
	loadProperties := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsByGroupInTransaction(tx, groupGranteeName, user, pagingRequest, loadPermissions, loadProperties)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjectsWithPropertiesByGroup", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

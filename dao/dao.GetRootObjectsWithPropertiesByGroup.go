package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetRootObjectsWithPropertiesByGroup retrieves a list of Objects and their
// Properties in Object Drive that are not nested beneath any other objects
// natively (natural parentId is null) and are owned by the specified group
func (dao *DataAccessLayer) GetRootObjectsWithPropertiesByGroup(groupGranteeName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsWithPropertiesByGroupInTransaction(tx, groupGranteeName, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjectsWithPropertiesByGroup", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsWithPropertiesByGroupInTransaction(tx *sqlx.Tx, groupGranteeName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response, err := getRootObjectsByGroupInTransaction(tx, groupGranteeName, user, pagingRequest)
	if err != nil {
		return response, err
	}
	// Fetch Properties
	for i := 0; i < len(response.Objects); i++ {
		dbProperties, dbPropErr := getPropertiesForObjectInTransaction(tx, response.Objects[i])
		response.Objects[i].Properties = dbProperties
		if dbPropErr != nil {
			err = dbPropErr
			return response, err
		}
	}

	return response, err
}

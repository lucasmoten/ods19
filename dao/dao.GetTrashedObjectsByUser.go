package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetTrashedObjectsByUser ...
func (dao *DataAccessLayer) GetTrashedObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetTrashedObjectsByUser")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	results, err := getTrashedObjectsByUserInTransaction(dao, tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetTrashedObjectsByUser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return results, err
}

func getTrashedObjectsByUserInTransaction(dao *DataAccessLayer, tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	loadProperties := true
	loadPermissions := true
	var response models.ODObjectResultset
	var err error

	query := `
    select 
        o.id    
    from object o
        inner join object_type ot on o.typeid = ot.id `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 1 and o.isExpunged = 0 and o.isAncestorDeleted = 0 `
	query += buildFilterRequireObjectsIOrMyGroupsOwn(tx, user)
	query += buildFilterSortAndLimit(pagingRequest)
	err = tx.Select(&response.Objects, query)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, queryRowCount(query))
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	// Load full meta, properties, and permissions
	for i := 0; i < len(response.Objects); i++ {
		obj, err := getObjectInTransaction(dao, tx, response.Objects[i], loadPermissions, loadProperties)
		if err != nil {
			return response, err
		}
		response.Objects[i] = obj
	}
	if loadProperties {
		response = postProcessingFilterOnCustomProperties(response, pagingRequest)
	}
	return response, err
}

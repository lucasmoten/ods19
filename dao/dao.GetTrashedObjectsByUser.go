package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetTrashedObjectsByUser ...
func (dao *DataAccessLayer) GetTrashedObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	results, err := getTrashedObjectsByUserInTransaction(tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetTrashedObjectsByUser", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return results, err
}

func getTrashedObjectsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	var response models.ODObjectResultset
	var err error

	query := `
    select 
        distinct sql_calc_found_rows 
        o.id    
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on op.objectid = o.id and op.isdeleted = 0 and op.allowread = 1 `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 1 and o.isExpunged = 0 and o.isAncestorDeleted = 0 `
	query += buildFilterRequireObjectsIOrMyGroupsOwn(tx, user)
	query += buildFilterForUserACMShare(tx, user)
	if !isOption409() {
		query += buildFilterForUserSnippets(user)
	}
	query += buildFilterSortAndLimit(pagingRequest)
	err = tx.Select(&response.Objects, query)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	// Load full meta, properties, and permissions
	for i := 0; i < len(response.Objects); i++ {
		obj, err := getObjectInTransaction(tx, response.Objects[i], true)
		if err != nil {
			return response, err
		}
		response.Objects[i] = obj
	}

	return response, err
}

package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// GetObjectsSharedToEveryone retrieves a list of Objects that have a permission that is sharing to everyone
func (dao *DataAccessLayer) GetObjectsSharedToEveryone(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectsSharedToEveryoneInTransaction(tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectsSharedToEveryone", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsSharedToEveryoneInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}

	// Get distinct due to multiple permissions may yield the same.
	// Only include those that are shared to everyone
	query := `
    select
        distinct sql_calc_found_rows 
        o.id    
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on op.objectId = o.id and op.isdeleted = 0 and op.allowread = 1 and op.grantee = '` + MySQLSafeString2(models.AACFlatten(models.EveryoneGroup)) + `' `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 `
	if !isOption409() {
		query += buildFilterExcludeNonRootedSharedToEveryone()
	}
	query += buildFilterForUserSnippets(user)
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query)
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

// buildFilterExcludeNonRootedSharedToEveryone builds a where clause portion
// for a sql statement suitable for filtering returned objects to not include
// those whose parent is also shared to everyone.
func buildFilterExcludeNonRootedSharedToEveryone() string {
	return `
	 and (
		o.parentId is null or o.parentId not in (
			select 
				objectId 
			from 
				object_permission
			where 
				isdeleted = 0 
				and allowRead = 1
				and grantee like '` + MySQLSafeString(models.AACFlatten(models.EveryoneGroup)) + `'
		)
	)
	`
}

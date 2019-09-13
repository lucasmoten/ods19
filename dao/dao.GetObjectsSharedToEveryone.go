package dao

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// GetObjectsSharedToEveryone retrieves a list of Objects that have a permission that is sharing to everyone
func (dao *DataAccessLayer) GetObjectsSharedToEveryone(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetObjectsSharedToEveryone")()
	loadProperties := true
	loadPermissions := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectsSharedToEveryoneInTransaction(tx, user, pagingRequest, loadPermissions, loadProperties)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectsSharedToEveryone", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsSharedToEveryoneInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, loadPermissions bool, loadProperties bool) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}

	// Only include those that are shared to everyone
	query := `
    select
        o.id    
    from object o
        inner join object_type ot on o.typeid = ot.id `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 `
	query += " and (acm2.flattenedacm like '%f_share=' or acm2.flattenedacm like '%f_share=;%')"
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query)
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
		obj, err := getObjectInTransaction(tx, response.Objects[i], loadPermissions, loadProperties)
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

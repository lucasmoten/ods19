package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// GetObjectsIHaveShared retrieves a list of Objects that I have explicitly
// shared to others
func (dao *DataAccessLayer) GetObjectsIHaveShared(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectsIHaveSharedInTransaction(tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectsIHaveShared", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsIHaveSharedInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}

	query := `
    select 
        distinct sql_calc_found_rows 
        o.id 
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on op.objectId = o.id and op.isdeleted = 0 and op.allowread = 1 `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 
        and op.createdby = '` + MySQLSafeString2(user.DistinguishedName) + `' 
		and op.grantee <> '` + MySQLSafeString2(models.AACFlatten(user.DistinguishedName)) + `'`
	if !isOption409() {
		query += buildFilterExcludeNonRootedIHaveShared(user)
		query += buildFilterForUserSnippets(user)
	}
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

// buildFilterExcludeNonRootedIHaveShared builds a where clause portion for a
// sql statement suitable for filtering returned objects to not include those
// whose parent is also shared by the user
func buildFilterExcludeNonRootedIHaveShared(user models.ODUser) string {
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
				and grantee <> '` + MySQLSafeString2(models.AACFlatten(user.DistinguishedName)) + `'
		)
	)
	`
}

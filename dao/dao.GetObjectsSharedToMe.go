package dao

import (
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
)

// GetObjectsSharedToMe retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetObjectsSharedToMe(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetObjectsSharedToMe")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectsSharedToMeInTransaction(tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectsSharedToMe", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsSharedToMeInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}

	// Get distinct due to multiple permissions may yield the same.
	// Filter out object owned by since owner's don't need to list items they've shared to themself
	// Only list explicit shares to avoid all nested children appearing in same list
	query := `
    select
        distinct sql_calc_found_rows 
        o.id    
    from object o
        inner join object_type ot on o.typeid = ot.id `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 `
	query += buildFilterExcludeObjectsIOrMyGroupsOwn(tx, user)
	// exclude those shared to everyone. for shared to me either explicit to me, or to a group im a member of
	query += " and (acm2.flattenedacm like '%f_share=%' and acm2.flattenedacm not like '%f_share=;%' and acm2.flattenedacm not like '%f_share=')"
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

// buildFilterExcludeNonRootedSharedToMe builds a where clause portion for a
// sql statement suitable for filtering returned objects to not include those
// those whose parent is also shared to the user as determined by the snippets
// associated with them as their f_share values containing groups and userdn
func buildFilterExcludeNonRootedSharedToMe(tx *sqlx.Tx, user models.ODUser) string {
	var sql string
	sql += " and (o.parentId is null or o.parentId not in ("
	sql += "select objectId from object_permission where isdeleted = 0 and allowRead = 1 and grantee in ("
	sql += "'" + strings.Join(getACMValueNamesForUser(tx, user, "f_share"), "','") + "'"
	sql += ")))"
	return sql
}

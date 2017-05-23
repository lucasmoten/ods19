package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// GetRootObjectsByGroup retrieves a list of Objects in Object Drive that are
// not nested beneath any other objects natively (natural parentId is null) and
// are owned by the specified group.
func (dao *DataAccessLayer) GetRootObjectsByGroup(groupGranteeName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsByGroupInTransaction(tx, groupGranteeName, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjectsByGroup", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsByGroupInTransaction(tx *sqlx.Tx, groupGranteeName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	// NOTE: While this looks similar to GetChildObjectsByUser there is more at
	// stake here as there is the requirement that the object permission grantee
	// is also the owner of each matching object.
	query := `
    select 
        distinct sql_calc_found_rows 
        o.id      
    from object o
        inner join object_type ot on o.typeid = ot.id `
	query += `inner join acm2 on o.acmid = acm2.id inner join useracm on acm2.id = useracm.acmid inner join user on useracm.userid = user.id and user.distinguishedname = '`
	query += MySQLSafeString(user.DistinguishedName)
	query += `'`
	query += ` where o.isdeleted = 0 and o.parentid is null `
	query += buildFilterRequireObjectsGroupOwns(tx, groupGranteeName)
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
		obj, err := getObjectInTransaction(tx, response.Objects[i], false)
		if err != nil {
			return response, err
		}
		response.Objects[i] = obj
	}
	return response, err
}

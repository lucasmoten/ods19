package dao

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// GetRootObjectsByUser retrieves a list of Objects in Object Drive that are
// not nested beneath any other objects natively (natural parentId is null) and
// are owned by the specified user.
func (dao *DataAccessLayer) GetRootObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetRootObjectsByUser")()
	loadProperties := true
	loadPermissions := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsByUserInTransaction(tx, user, pagingRequest, loadPermissions, loadProperties)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjectsByUser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, loadPermissions bool, loadProperties bool) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	// NOTE: While this looks similar to GetChildObjectsByUser there is more at
	// stake here as there is the requirement that the object permission grantee
	// is also the owner of each matching object.
	query := `
    select 
        o.id        
    from object o
        inner join object_type ot on o.typeid = ot.id `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 and o.parentid is null `
	query += buildFilterRequireObjectsIOwn(tx, user)
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

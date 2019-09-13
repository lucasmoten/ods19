package dao

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// GetChildObjectsByUser retrieves a list of Objects in Object Drive that are
// nested beneath a specified object by parentID and are owned by the specified
// user or group.
func (dao *DataAccessLayer) GetChildObjectsByUser(
	user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	defer util.Time("GetChildObjectsByUser")()
	loadPermissions := true
	loadProperties := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getChildObjectsByUserInTransaction(tx, user, pagingRequest, object, loadPermissions, loadProperties)
	if err != nil {
		dao.GetLogger().Error("error in getchildobjectsbyuser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, object models.ODObject, loadPermissions bool, loadProperties bool) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	query := `
    select 
        o.id    
	from object o
        inner join object_type ot on o.typeid = ot.id `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 and o.parentid = ? `
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query, object.ID)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, queryRowCount(query), object.ID)
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

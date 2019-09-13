package dao

import (
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// GetObjectsIHaveShared retrieves a list of Objects that I have explicitly
// shared to others
func (dao *DataAccessLayer) GetObjectsIHaveShared(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetObjectsIHaveShared")()
	loadPermissions := true
	loadProperties := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectsIHaveSharedInTransaction(tx, user, pagingRequest, loadPermissions, loadProperties)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectsIHaveShared", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsIHaveSharedInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, loadPermissions bool, loadProperties bool) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}

	query := `
    select 
        distinct sql_calc_found_rows 
        o.id 
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on op.objectId = o.id and op.isdeleted = 0 and op.allowread = 1 `
	query += buildJoinUserToACM(tx, user)
	query += ` where o.isdeleted = 0 `
	usergranteeid := strconv.FormatInt(getACMValueFor(tx, models.AACFlatten(user.DistinguishedName)), 10)
	query += fmt.Sprintf(` and op.createdbyid = %s and op.granteeid <> %s `, usergranteeid, usergranteeid)
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

package dao

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
)

// GetRootObjects retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetRootObjects(pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	defer util.Time("GetRootObjects")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsInTransaction(tx, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjects", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsInTransaction(tx *sqlx.Tx, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	query := `
    select 
        o.id    
    from object o 
        inner join object_type ot on o.typeid = ot.id
    where o.isdeleted = 0 and o.parentid is null`
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
		obj, err := getObjectInTransaction(tx, response.Objects[i], true)
		if err != nil {
			return response, err
		}
		response.Objects[i] = obj
	}
	return response, err
}

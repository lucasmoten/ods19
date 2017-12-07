package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
)

// GetChildObjects retrieves a list of Objects in Object Drive that are nested
// beneath a specified object by parentID
func (dao *DataAccessLayer) GetChildObjects(pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	defer util.Time("GetChildObjects")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getChildObjectsInTransaction(tx, pagingRequest, object, false)
	if err != nil {
		dao.GetLogger().Error("Error in GetChildObjects", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsInTransaction(tx *sqlx.Tx, pagingRequest PagingRequest, object models.ODObject, loadProperties bool) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	query := `
    select 
        o.id    
    from object o 
        inner join object_type ot on o.typeid = ot.id 
    where o.isdeleted = 0 and o.parentid = ?`
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
		obj, err := getObjectInTransaction(tx, response.Objects[i], loadProperties)
		if err != nil {
			return response, err
		}
		response.Objects[i] = obj
	}
	return response, err
}

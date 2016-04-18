package dao

import (
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// GetRootObjects retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetRootObjects(pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getRootObjectsInTransaction(tx, pagingRequest)
	if err != nil {
		log.Printf("Error in GetRootObjects: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsInTransaction(tx *sqlx.Tx, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	query := `select sql_calc_found_rows o.*, ot.name typeName
        from object o 
            inner join object_type ot on o.typeid = ot.id
        where 
            o.isdeleted = 0 and o.parentid is null`
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query)
	if err != nil {
		print(err.Error())
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	for i := 0; i < len(response.Objects); i++ {
		permissions, err := getPermissionsForObjectInTransaction(tx, response.Objects[i])
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Permissions = permissions
	}
	return response, err
}

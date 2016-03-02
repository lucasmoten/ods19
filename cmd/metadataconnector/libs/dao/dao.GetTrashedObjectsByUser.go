package dao

import (
	"log"
	"strconv"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetTrashedObjectsByUser ...
func (dao *DataAccessLayer) GetTrashedObjectsByUser(orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
	var results models.ODObjectResultset
	var err error
	tx := dao.MetadataDB.MustBegin()
	results, err = getTrashedObjectsByUserInTransaction(tx, orderByClause, pageNumber, pageSize, user)
	if err != nil {
		log.Printf("Error in GetTrashedObjectsByUser: %v\n", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return results, err
}

func getTrashedObjectsByUserInTransaction(tx *sqlx.Tx, orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
	var response models.ODObjectResultset
	var err error
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)

	query := `select distinct sql_calc_found_rows o.*, ot.name typeName
        from object o
        inner join object_type ot
            on o.typeid = ot.id
        inner join object_permission op
            on o.id = op.objectid
        and op.isdeleted = 0
        and op.allowread = 1
        where o.isdeleted = 1 and o.ownedBy = ? and o.isExpunged = 0
        and o.isAncestorDeleted = 0 `
	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err = tx.Select(&response.Objects, query, user)
	if err != nil {
		print(err.Error())
	}
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = GetSanitizedPageNumber(pageNumber)
	response.PageSize = GetSanitizedPageSize(pageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	return response, err
}

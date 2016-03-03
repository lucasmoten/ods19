package dao

import (
	"log"
	"strconv"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// GetObjectsIHaveShared retrieves a list of Objects that I have explicitly
// shared to others
func (dao *DataAccessLayer) GetObjectsIHaveShared(orderByClause string,
	pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getObjectsIHaveSharedInTransaction(tx, orderByClause, pageNumber, pageSize, user)
	if err != nil {
		log.Printf("Error in GetObjectsIHaveShared: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsIHaveSharedInTransaction(tx *sqlx.Tx, orderByClause string,
	pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)

	//Note: not quite right, because we need to also join in allowUpdate, etc.
	query := `select distinct sql_calc_found_rows o.*, ot.name typeName 
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on op.objectId = o.id
    where o.isdeleted = 0 
        and op.isdeleted = 0 
        and op.explicitShare = 1
        and op.createdBy = ?
        and op.grantee <> ?
  `

	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := tx.Select(&response.Objects, query, user, user)
	if err != nil {
		return response, err
	}
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pageNumber)
	response.PageSize = GetSanitizedPageSize(pageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	return response, err
}

package dao

import (
	"strconv"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetRootObjects retrieves a list of Objects in Object Drive that are not
// nested beneath any other objects natively (natural parentId is null)
func GetRootObjects(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows o.*, ot.name typeName from object o inner join object_type ot on o.typeid = ot.id where o.isdeleted = 0 and o.parentid is null`
	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := db.Select(&response.Objects, query)
	if err != nil {
		print(err.Error())
	}
	err = db.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = pageNumber
	response.PageSize = pageSize
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, pageSize)
	return response, err
}

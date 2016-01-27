package dao

import (
	"strconv"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetChildObjects retrieves a list of Objects in Object Drive that are nested
// beneath a specified object by parentID
func GetChildObjects(db *sqlx.DB, orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows * from object where isdeleted = 0 and parentid = ?`
	if len(orderByClause) > 0 {
		query += ` order by ` + orderByClause
	} else {
		query += ` order by createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := db.Select(&response.Objects, query, object.ID)
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

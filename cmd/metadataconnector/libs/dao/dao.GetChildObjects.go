package dao

import (
	"strconv"

	"decipher.com/oduploader/metadata/models"
)

// GetChildObjects retrieves a list of Objects in Object Drive that are nested
// beneath a specified object by parentID
func (dao *DataAccessLayer) GetChildObjects(
	orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows o.*, ot.name typeName from object o inner join object_type ot on o.typeid = ot.id where o.isdeleted = 0 and o.parentid = ?`
	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := dao.MetadataDB.Select(&response.Objects, query, object.ID)
	if err != nil {
		print(err.Error())
	}
	err = dao.MetadataDB.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = pageNumber
	response.PageSize = pageSize
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, pageSize)
	return response, err

}

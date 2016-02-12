package dao

import (
	"strconv"

	"decipher.com/oduploader/metadata/models"
)

// GetRootObjectsByOwner retrieves a list of Objects in Object Drive that are
// not nested beneath any other objects natively (natural parentId is null) and
// are owned by the specified user or group.
func (dao *DataAccessLayer) GetRootObjectsByOwner(
	orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows o.*, ot.name typeName from object o inner join object_type ot on o.typeid = ot.id where o.isdeleted = 0 and o.parentid is null and o.ownedby = ?`
	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := dao.MetadataDB.Select(&response.Objects, query, owner)
	if err != nil {
		print(err.Error())
	}
	// TODO: This relies on sql_calc_found_rows from previous call, but I dont know if its guaranteed that the reference to db here
	// for this call would be the same as that used above from the built in connection pooling perspective.  If it isn't, then it
	// could conceivably get the result from a concurrent instance performing a similar operation.
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

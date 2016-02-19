package dao

import (
	"strconv"

	"decipher.com/oduploader/metadata/models"
)

// GetObjectsSharedToMe retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetObjectsSharedToMe(
	owner string,
	orderByClause string,
	pageNumber int,
	pageSize int,
) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)

	//Note: not quite right, because we need to also join in allowUpdate, etc.
	query := `
  select
    sql_calc_found_rows o.*,
    ot.name typeName
  from object o
  inner join object_type ot on o.typeid = ot.id
  inner join object_permission op on op.objectId = o.id
  where
    o.isdeleted = 0 and
    o.parentid is null and
    op.grantee = ? and
    op.createdBy <> ?
  `

	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := dao.MetadataDB.Select(&response.Objects, query, owner, owner)
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

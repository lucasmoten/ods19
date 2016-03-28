package dao

import (
	"log"
	"strconv"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// GetRootObjects retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetRootObjects(orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getRootObjectsInTransaction(tx, orderByClause, pageNumber, pageSize)
	if err != nil {
		log.Printf("Error in GetRootObjects: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsInTransaction(tx *sqlx.Tx, orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)
	query := `select sql_calc_found_rows o.*, ot.name typeName
  from object o inner join object_type ot on o.typeid = ot.id
  where o.isdeleted = 0 and o.parentid is null`
	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := tx.Select(&response.Objects, query)
	if err != nil {
		print(err.Error())
		return response, err
	}
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pageNumber)
	response.PageSize = GetSanitizedPageSize(pageSize)
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

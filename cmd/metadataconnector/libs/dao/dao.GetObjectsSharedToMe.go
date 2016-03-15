package dao

import (
	"log"
	"strconv"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// GetObjectsSharedToMe retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetObjectsSharedToMe(
	grantee string,
	orderByClause string,
	pageNumber int,
	pageSize int,
) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getObjectsSharedToMeInTransaction(tx, grantee, orderByClause, pageNumber, pageSize)
	if err != nil {
		log.Printf("Error in GetObjectsSharedTome: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsSharedToMeInTransaction(tx *sqlx.Tx, grantee string,
	orderByClause string,
	pageNumber int,
	pageSize int,
) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)

	// Get distinct due to multiple permissions may yield the same.
	// Filter out object owned by since owner's don't need to list items they've shared to themself
	// Only list explicit shares to avoid all nested children appearing in same list
	query := `
        select distinct
            sql_calc_found_rows o.*,
            ot.name typeName
        from object o
            inner join object_type ot on o.typeid = ot.id
            inner join object_permission op on op.objectId = o.id
        where
            op.isdeleted = 0 and
            op.allowread = 1 and
            op.explicitshare = 1 and
            op.grantee = ? and
            o.isdeleted = 0 and
            o.ownedBy <> ?
        `

	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := tx.Select(&response.Objects, query, grantee, grantee)
	if err != nil {
		print(err.Error())
	}
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = pageNumber
	response.PageSize = pageSize
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, pageSize)
	return response, err
}

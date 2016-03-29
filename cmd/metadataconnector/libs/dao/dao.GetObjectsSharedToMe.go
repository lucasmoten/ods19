package dao

import (
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

// GetObjectsSharedToMe retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetObjectsSharedToMe(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getObjectsSharedToMeInTransaction(tx, user, pagingRequest)
	if err != nil {
		log.Printf("Error in GetObjectsSharedTome: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsSharedToMeInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}

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
            o.ownedBy <> ? `
	query += buildFilterForUserACM(user)
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query, user.DistinguishedName, user.DistinguishedName)
	if err != nil {
		print(err.Error())
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	return response, err
}

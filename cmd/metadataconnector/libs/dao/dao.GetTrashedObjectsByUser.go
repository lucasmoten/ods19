package dao

import (
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"github.com/jmoiron/sqlx"
)

// GetTrashedObjectsByUser ...
func (dao *DataAccessLayer) GetTrashedObjectsByUser(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	var results models.ODObjectResultset
	var err error
	tx := dao.MetadataDB.MustBegin()
	results, err = getTrashedObjectsByUserInTransaction(tx, user, pagingRequest)
	if err != nil {
		log.Printf("Error in GetTrashedObjectsByUser: %v\n", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return results, err
}

func getTrashedObjectsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	var response models.ODObjectResultset
	var err error

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
	query += buildFilterForUserACM(user)
	query += buildFilterSortAndLimit(pagingRequest)
	err = tx.Select(&response.Objects, query, user.DistinguishedName)
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

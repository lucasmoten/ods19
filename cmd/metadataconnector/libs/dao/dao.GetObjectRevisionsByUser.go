package dao

import (
	"log"
	"strconv"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// GetObjectRevisionsByUser retrieves a list of revisions for an object.
func (dao *DataAccessLayer) GetObjectRevisionsByUser(
	orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getObjectRevisionsByUserInTransaction(tx, orderByClause, pageNumber, pageSize, object, user)
	if err != nil {
		log.Printf("Error in GetObjectRevisionsByUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectRevisionsByUserInTransaction(tx *sqlx.Tx, orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)
	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	// TODO: Incorporate support for ACM checks. This may need to be passed as
	// an argument as additional whereByClause to avoid complex coupling
	query := `select distinct sql_calc_found_rows 
        ao.id, ao.createdDate, ao.createdBy, ao.modifiedDate, ao.modifiedBy, ao.isDeleted, ao.deletedDate, ao.deletedBy,
        ao.isAncestorDeleted, ao.isExpunged, ao.expungedDate, ao.expungedBy, ao.changeCount, ao.changeToken, 
        ao.ownedBy, ao.typeId, ao.name, ao.description, ao.parentId, ao.contentConnector, ao.rawAcm, ao.contentType,
        ao.contentSize, ao.contentHash, ao.encryptIV,
            ot.name typeName
        from a_object ao
        inner join object_type ot
            on ao.typeid = ot.id
        inner join object_permission op
            on ao.id = op.objectid
            and op.isdeleted = 0
            and op.allowread = 1
        where ao.isexpunged = 0 
            and ao.id = ? 
            and op.grantee = ?`
	if len(orderByClause) > 0 {
		query += ` order by ao.` + orderByClause
	} else {
		query += ` order by ao.modifieddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := tx.Select(&response.Objects, query, object.ID, user)
	if err != nil {
		return response, err
	}
	// This relies on sql_calc_found_rows in previous call and must be done within
	// a transaction to maintain context
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pageNumber)
	response.PageSize = GetSanitizedPageSize(pageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	// TODO: Permissions based on current state.. this would only be worth doing if
	// it could get permissions based on revision and also restrict the list which hits
	// the same but may be confusing as it could leave some revisions hidden from users
	//
	// for i := 0; i < len(response.Objects); i++ {
	// 	permissions, err := getPermissionsForObjectRevisionInTransaction(tx, response.Objects[i])
	// 	if err != nil {
	// 		print(err.Error())
	// 		return response, err
	// 	}
	// 	response.Objects[i].Permissions = permissions
	// }
	return response, err
}

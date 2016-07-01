package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// GetObjectRevisionsByUser retrieves a list of revisions for an object.
func (dao *DataAccessLayer) GetObjectRevisionsByUser(
	user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectRevisionsByUserInTransaction(tx, user, pagingRequest, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectRevisionsByUser", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectRevisionsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	query := `
    select 
        distinct sql_calc_found_rows
        ao.id
        ,ao.createdDate
        ,ao.createdBy
        ,ao.modifiedDate
        ,ao.modifiedBy
        ,ao.isDeleted
        ,ao.deletedDate
        ,ao.deletedBy
        ,ao.isAncestorDeleted
        ,ao.isExpunged
        ,ao.expungedDate
        ,ao.expungedBy
        ,ao.changeCount
        ,ao.changeToken
        ,ao.ownedBy
        ,ao.typeId
        ,ao.name
        ,ao.description
        ,ao.parentId
        ,ao.contentConnector
        ,ao.rawAcm
        ,ao.contentType
        ,ao.contentSize
        ,ao.contentHash
        ,ao.encryptIV
        ,ao.ownedByNew
        ,ao.isPDFAvailable
        ,ao.isStreamStored
        ,ao.isUSPersonsData
        ,ao.isFOIAExempt
        ,ot.name typeName
    from a_object ao 
        inner join object_type ot on ao.typeid = ot.id
        inner join object_permission op on ao.id = op.objectid and op.isdeleted = 0 and op.allowread = 1
        inner join objectacm acm on ao.id = acm.objectid
    where 
        ao.isexpunged = 0
        and ao.id = ? 
    `
	query += buildFilterForUserACMShare(user)
	query += buildFilterForUserSnippets(user)
	query += buildFilterSortAndLimitArchive(pagingRequest)
	err := tx.Select(&response.Objects, query, object.ID, user.DistinguishedName)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
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

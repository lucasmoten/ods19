package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// GetObjectsIHaveShared retrieves a list of Objects that I have explicitly
// shared to others
func (dao *DataAccessLayer) GetObjectsIHaveShared(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectsIHaveSharedInTransaction(tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectsIHaveShared", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsIHaveSharedInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}

	query := `
    select 
        distinct sql_calc_found_rows 
        o.id    
        ,o.createdDate
        ,o.createdBy
        ,o.modifiedDate
        ,o.modifiedBy
        ,o.isDeleted
        ,o.deletedDate
        ,o.deletedBy
        ,o.isAncestorDeleted
        ,o.isExpunged
        ,o.expungedDate
        ,o.expungedBy
        ,o.changeCount
        ,o.changeToken
        ,o.ownedBy
        ,o.typeId
        ,o.name
        ,o.description
        ,o.parentId
        ,o.contentConnector
        ,o.rawAcm
        ,o.contentType
        ,o.contentSize
        ,o.contentHash
        ,o.encryptIV
        ,o.ownedByNew
        ,o.isPDFAvailable
        ,o.isStreamStored
        ,o.containsUSPersonsData
        ,o.exemptFromFOIA        
        ,ot.name typeName    
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on op.objectId = o.id
        inner join objectacm acm on o.id = acm.objectid            
    where 
        o.isdeleted = 0 
        and op.isdeleted = 0 
        and op.createdBy = ?
        and op.grantee <> ? `
	query += buildFilterForUserSnippets(user)
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query, user.DistinguishedName, user.DistinguishedName)
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
	// Load permissions
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

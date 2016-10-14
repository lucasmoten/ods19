package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// GetObjectsSharedToMe retrieves a list of Objects that are not nested
// beneath any other objects natively (natural parentId is null).
func (dao *DataAccessLayer) GetObjectsSharedToMe(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectsSharedToMeInTransaction(tx, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectsSharedToMe", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectsSharedToMeInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}

	// Get distinct due to multiple permissions may yield the same.
	// Filter out object owned by since owner's don't need to list items they've shared to themself
	// Only list explicit shares to avoid all nested children appearing in same list
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
        op.isdeleted = 0 
        and op.allowread = 1 
        and o.isdeleted = 0 
        and o.ownedBy <> ? `

	query += buildFilterExcludeEveryone()
	query += buildFilterExcludeNonRootedShares(user)
	query += buildFilterForUserACMShare(user)
	query += buildFilterForUserSnippets(user)
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

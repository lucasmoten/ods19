package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// GetRootObjectsByGroup retrieves a list of Objects in Object Drive that are
// not nested beneath any other objects natively (natural parentId is null) and
// are owned by the specified group.
func (dao *DataAccessLayer) GetRootObjectsByGroup(groupName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getRootObjectsByGroupInTransaction(tx, groupName, user, pagingRequest)
	if err != nil {
		dao.GetLogger().Error("Error in GetRootObjectsByUser", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsByGroupInTransaction(tx *sqlx.Tx, groupName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	// NOTE: While this looks similar to GetChildObjectsByUser there is more at
	// stake here as there is the requirement that the object permission grantee
	// is also the owner of each matching object.
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
        and op.allowread = 1
        and o.parentid is null `
	query += buildFilterRequireObjectsGroupOwns(tx, groupName)
	query += buildFilterForUserACMShare(user)
	query += buildFilterForUserSnippets(user)
	query += buildFilterSortAndLimit(pagingRequest)

	//log.Println(query)
	err := tx.Select(&response.Objects, query)
	if err != nil {
		print(err.Error())
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
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

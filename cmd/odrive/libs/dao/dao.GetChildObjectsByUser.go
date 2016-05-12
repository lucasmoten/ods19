package dao

import (
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// GetChildObjectsByUser retrieves a list of Objects in Object Drive that are
// nested beneath a specified object by parentID and are owned by the specified
// user or group.
func (dao *DataAccessLayer) GetChildObjectsByUser(
	user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getChildObjectsByUserInTransaction(tx, user, pagingRequest, object)
	if err != nil {
		log.Printf("Error in GetChildObjectsByUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getChildObjectsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}

	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
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
        ,o.isUSPersonsData
        ,o.isFOIAExempt        
        ,ot.name typeName    
	from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on o.id = op.objectid and op.isdeleted = 0 and op.allowread = 1
        inner join object_acm acm on o.id = acm.objectid            
    where 
        o.isdeleted = 0 
        and o.parentid = ?`
	query += buildFilterForUserACMShare(user)
	query += buildFilterForUserACM(user)
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query, object.ID, user.DistinguishedName)
	if err != nil {
		print(err.Error())
		return response, err
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

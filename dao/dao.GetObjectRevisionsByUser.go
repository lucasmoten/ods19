package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// GetObjectRevisionsByUser retrieves a list of revisions for an object.
func (dao *DataAccessLayer) GetObjectRevisionsByUser(
	user models.ODUser, pagingRequest PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectRevisionsByUserInTransaction(tx, user, pagingRequest, object, checkACM)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectRevisionsByUser", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectRevisionsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error) {
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
        ,ao.containsUSPersonsData
        ,ao.exemptFromFOIA
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

	// Redact by ACM access and set permissions
	permissions := []models.ODObjectPermission{}
	for i, o := range response.Objects {
		ok := checkACM(&o)
		if !ok {
			//Preserve the id field and list the changeCount field
			id := response.Objects[i].ID
			response.Objects[i] = models.ODObject{}
			response.Objects[i].ID = id
			response.Objects[i].ChangeCount = -1
		}
		if len(permissions) == 0 {
			permissions, err = getPermissionsForObjectInTransaction(tx, object)
			if err != nil {
				return response, err
			}
		}
		response.Objects[i].Permissions = permissions
	}

	return response, err
}

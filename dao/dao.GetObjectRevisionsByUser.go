package dao

import (
	"strings"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
)

// GetObjectRevisionsByUser retrieves a list of revisions for an object.
func (dao *DataAccessLayer) GetObjectRevisionsByUser(
	user models.ODUser, pagingRequest PagingRequest, object models.ODObject, withProperties bool) (models.ODObjectResultset, error) {
	defer util.Time("GetObjectRevisionsByUser")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectResultset{}, err
	}
	response, err := getObjectRevisionsByUserInTransaction(tx, user, pagingRequest, object, withProperties)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectRevisionsByUser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getObjectRevisionsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest, object models.ODObject, withProperties bool) (models.ODObjectResultset, error) {
	response := models.ODObjectResultset{}
	query := `
    select 
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
        ,ao.containsUSPersonsData
        ,ao.exemptFromFOIA
        ,ot.name typeName
		,ao.acmid
    from a_object ao 
        inner join object_type ot on ao.typeid = ot.id `
	query += buildJoinUserToACM(tx, user)
	query += ` where ao.isdeleted = 0 and ao.id = ? `
	query += buildFilterSortAndLimitArchive(pagingRequest)
	query = strings.Replace(query, "a_object ao", "a_object o", -1)
	query = strings.Replace(query, "ao.", "o.", -1)
	err := tx.Select(&response.Objects, query, object.ID)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, queryRowCount(query), object.ID)
	if err != nil {
		return response, err
	}
	response.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
	response.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)

	// Get detail information for each object
	permissions := []models.ODObjectPermission{}
	for i := 0; i < len(response.Objects); i++ {
		// Populate properties if requested
		if withProperties {
			properties, err := getPropertiesForObjectRevisionInTransaction(tx, response.Objects[i])
			if err != nil {
				return response, err
			}
			response.Objects[i].Properties = properties
		}
		// Permissions
		if len(permissions) == 0 {
			// Not yet retrieved, do it now
			permissions, err = getPermissionsForObjectInTransaction(tx, object)
			if err != nil {
				return response, err
			}
		}
		response.Objects[i].Permissions = permissions
	}
	return response, err
}

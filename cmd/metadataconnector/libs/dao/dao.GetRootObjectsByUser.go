package dao

import (
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// GetRootObjectsByUser retrieves a list of Objects in Object Drive that are
// not nested beneath any other objects natively (natural parentId is null) and
// are owned by the specified user or group.
func (dao *DataAccessLayer) GetRootObjectsByUser(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getRootObjectsByUserInTransaction(tx, user, pagingRequest)
	if err != nil {
		log.Printf("Error in GetRootObjectsByUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	// NOTE: While this looks similar to GetChildObjectsByUser there is more at
	// stake here as there is the requirement that the object permission grantee
	// is also the owner of each matching object.
	query := `select distinct sql_calc_found_rows o.*, ot.name typeName
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on o.id = op.objectid
            and op.isdeleted = 0
            and op.allowread = 1
            and o.ownedBy = op.grantee
    where 
        o.isdeleted = 0 and o.parentid is null and o.ownedby = ? `
	query += buildFilterForUserACM(user)
	query += buildFilterSortAndLimit(pagingRequest)
	err := tx.Select(&response.Objects, query, user.DistinguishedName)
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

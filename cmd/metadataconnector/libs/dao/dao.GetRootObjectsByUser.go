package dao

import (
	"strconv"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// GetRootObjectsByUser retrieves a list of Objects in Object Drive that are
// not nested beneath any other objects natively (natural parentId is null) and
// are owned by the specified user or group.
func (dao *DataAccessLayer) GetRootObjectsByUser(
	orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := getRootObjectsByUserInTransaction(tx, orderByClause, pageNumber, pageSize, user)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getRootObjectsByUserInTransaction(tx *sqlx.Tx, orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {

	response := models.ODObjectResultset{}
	limit := GetLimit(pageNumber, pageSize)
	offset := GetOffset(pageNumber, pageSize)
	// NOTE: distinct is unfortunately used here because object_permission
	// allows multiple records per object and grantee.
	// NOTE: While this looks similar to GetChildObjectsByUser there is more at
	// stake here as there is the requirement that the object permission grantee
	// is also the owner of each matching object.
	// TODO: Incorporate support for ACM checks. This may need to be passed as
	// an argument as additional whereByClause to avoid complex coupling
	query := `select distinct sql_calc_found_rows o.*, ot.name typeName
	from object o
  inner join object_type ot
		on o.typeid = ot.id
	inner join object_permission op
		on o.id = op.objectid
		and op.isdeleted = 0
		and op.allowread = 1
		and o.ownedBy = op.grantee
  where o.isdeleted = 0 and o.parentid is null and o.ownedby = ?`
	if len(orderByClause) > 0 {
		query += ` order by o.` + orderByClause
	} else {
		query += ` order by o.createddate desc`
	}
	query += ` limit ` + strconv.Itoa(limit) + ` offset ` + strconv.Itoa(offset)
	err := tx.Select(&response.Objects, query, user)
	if err != nil {
		print(err.Error())
	}
	// TODO: This relies on sql_calc_found_rows from previous call, but I dont know if its guaranteed that the reference to db here
	// for this call would be the same as that used above from the built in connection pooling perspective.  If it isn't, then it
	// could conceivably get the result from a concurrent instance performing a similar operation.
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		print(err.Error())
	}
	response.PageNumber = GetSanitizedPageNumber(pageNumber)
	response.PageSize = GetSanitizedPageSize(pageSize)
	response.PageRows = len(response.Objects)
	response.PageCount = GetPageCount(response.TotalRows, response.PageSize)
	for i := 0; i < len(response.Objects); i++ {
		permissions, err := getPermissionsForObjectInTransaction(tx, &response.Objects[i])
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Permissions = permissions
	}
	return response, err
}

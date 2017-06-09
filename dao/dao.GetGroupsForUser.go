package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
)

// GetGroupsForUser retrieves a list of groups the user is a member of that have root objects and their counts
func (dao *DataAccessLayer) GetGroupsForUser(user models.ODUser) (models.GroupSpaceResultset, error) {
	defer util.Time("GetGroupsForUser")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.GroupSpaceResultset{}, err
	}
	response, err := getGroupsForUserInTransaction(tx, user)
	if err != nil {
		dao.GetLogger().Error("Error in GetGroupsForUser", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getGroupsForUserInTransaction(tx *sqlx.Tx, user models.ODUser) (models.GroupSpaceResultset, error) {

	response := models.GroupSpaceResultset{}

	sql := `
		select 
			ag.grantee grantee, 
			ag.resourceString resourceString, 
			ag.displayName displayName, 
			count(o.id) quantity 
		from 
			object o 
			inner join acmvalue2 av on o.ownedbyid = av.id 
			inner join useraocachepart uaocp on av.id = uaocp.uservalueid 
			inner join acmkey2 ak on uaocp.userkeyid = ak.id 
			inner join user u on uaocp.userid = u.id 
			inner join acmgrantee ag on av.name = ag.grantee 
		where 
			ak.name = 'f_share' 
			and u.distinguishedName = '` + MySQLSafeString2(user.DistinguishedName) + `'
			and ag.userDistinguishedName is null
			and o.parentid is null 
		group by av.name
		order by ag.displayName
	`

	err := tx.Select(&response.GroupSpaces, sql)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, "select found_rows()")
	if err != nil {
		return response, err
	}
	response.PageNumber = 1
	response.PageSize = response.TotalRows
	response.PageRows = response.TotalRows
	response.PageCount = 1
	return response, err
}

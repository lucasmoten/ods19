package dao

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// GetGroupsForUser retrieves a list of groups the user is a member of that have root objects and their counts
func (dao *DataAccessLayer) GetGroupsForUser(user models.ODUser) (models.GroupSpaceResultset, error) {
	defer util.Time("GetGroupsForUser")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.GroupSpaceResultset{}, err
	}
	response, err := getGroupsForUserInTransaction(tx, user)
	if err != nil {
		dao.GetLogger().Error("error in getgroupsforuser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getGroupsForUserInTransaction(tx *sqlx.Tx, user models.ODUser) (models.GroupSpaceResultset, error) {

	response := models.GroupSpaceResultset{}

	query := `
		select 
			distinct sql_calc_found_rows
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

	err := tx.Select(&response.GroupSpaces, query)
	if err != nil {
		return response, err
	}
	// Paging stats guidance
	err = tx.Get(&response.TotalRows, queryRowCount(query))
	if err != nil {
		return response, err
	}
	response.PageNumber = 1
	response.PageSize = response.TotalRows
	response.PageRows = response.TotalRows
	response.PageCount = 1
	return response, err
}

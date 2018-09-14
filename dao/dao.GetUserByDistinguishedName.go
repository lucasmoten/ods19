package dao

import (
	"database/sql"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func (dao *DataAccessLayer) GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error) {
	defer util.Time("GetUserByDistinguishedName")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODUser{}, err
	}
	dbUser, err := getUserByDistinguishedNameInTransaction(tx, user)
	if err != nil {
		if err != sql.ErrNoRows {
			dao.GetLogger().Error("Error in GetUserByDistinguishedName", zap.Error(err))
		}
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbUser, err
}

func getUserByDistinguishedNameInTransaction(tx *sqlx.Tx, user models.ODUser) (models.ODUser, error) {
	var dbUser models.ODUser
	getUserStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,changeCount
        ,changeToken
        ,distinguishedName
        ,displayName
        ,email
    from user 
    where distinguishedName = ?`
	err := tx.Get(&dbUser, getUserStatement, user.DistinguishedName)
	return dbUser, err
}

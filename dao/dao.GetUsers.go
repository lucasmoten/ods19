package dao

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetUsers retrieves all users.
func (dao *DataAccessLayer) GetUsers() ([]models.ODUser, error) {
	defer util.Time("GetUsers")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return []models.ODUser{}, err
	}
	result, err := getUsersInTransaction(dao.GetLogger(), tx)
	if err != nil {
		dao.GetLogger().Error("Error in GetUsers", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return result, err
}

func getUsersInTransaction(logger zap.Logger, tx *sqlx.Tx) ([]models.ODUser, error) {

	var result []models.ODUser
	getUsersStatement := `
    select 
        distinguishedName 
        ,displayName
        ,email 
    from user`
	err := tx.Select(&result, getUsersStatement)
	if err != nil {
		logger.Error("Unable to execute query", zap.String("sql", getUsersStatement), zap.String("err", err.Error()))
	}
	return result, err
}

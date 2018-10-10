package dao

import (
	"database/sql"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetDBState retrieves the database state including schema version and identifier used for cache location
func (dao *DataAccessLayer) GetDBState() (models.DBState, error) {
	dao.GetLogger().Debug("dao starting txn for GetDBState", zap.Int("open-connections before check", dao.GetOpenConnections()))
	defer util.Time("GetDBState")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.DBState{}, err
	}
	dao.GetLogger().Debug("dao passing  txn into getDBStateInTransaction")
	dbState, err := getDBStateInTransaction(tx)
	dao.GetLogger().Debug("dao returned txn from getDBStateInTransaction")
	if err != nil {
		if err != sql.ErrNoRows {
			dao.GetLogger().Warn("error in getdbstate", zap.Error(err))
		}
		dao.GetLogger().Debug("dao rolling back txn for GetDBState")
		tx.Rollback()
	} else {
		dao.GetLogger().Debug("dao committing txn for GetDBState")
		tx.Commit()
	}
	dao.GetLogger().Debug("dao finished txn for GetDBState")
	return dbState, err
}

func getDBStateInTransaction(tx *sqlx.Tx) (models.DBState, error) {
	var dbState models.DBState

	getDBStateStatement := `select createdDate, modifiedDate, schemaVersion, identifier from dbstate`
	err := tx.Unsafe().Get(&dbState, getDBStateStatement)
	if err != nil {
		return dbState, err
	}

	return dbState, nil
}

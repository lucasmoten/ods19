package dao

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetDBState retrieves the database state including schema version and identifier used for cache location
func (dao *DataAccessLayer) GetDBState() (models.DBState, error) {
	defer util.Time("GetDBState")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.String("err", err.Error()))
		return models.DBState{}, err
	}
	dbState, err := getDBStateInTransaction(tx)
	if err != nil {
		dao.GetLogger().Warn("error in getdbstate", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
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

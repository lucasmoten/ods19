package dao

import (
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetDBState retrieves the database state including schema version and identifier used for cache location
func (dao *DataAccessLayer) GetDBState() (models.DBState, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.DBState{}, err
	}
	dbState, err := getDBStateInTransaction(tx)
	if err != nil {
		dao.GetLogger().Error("Error in GetDBState", zap.String("err", err.Error()))
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

	// Warn if the version reported by DB doesn't match value set here in DAO
	if dbState.SchemaVersion != SchemaVersion {
		msg := "WARNING: Schema mismatch. Database is at version '%s' and DAO expects version '%s'"
		log.Printf(msg, dbState.SchemaVersion, SchemaVersion)
	}

	return dbState, nil
}

package dao_test

import (
	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
)

var db *sqlx.DB

func init() {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	var err error
	db, err = dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err)
	}
}

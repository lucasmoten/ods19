package dao

import (
	"fmt"
	"log"
	"time"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

func (dao *DataAccessLayer) GetDBState() (models.DBState, error) {

	tx := dao.MetadataDB.MustBegin()
	dbState, err := getDBStateInTransaction(tx)
	if err != nil {
		log.Printf("Error in GetDBState: %v\n", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbState, err
}

func getDBStateInTransaction(tx *sqlx.Tx) (models.DBState, error) {
	var dbState models.DBState

	getDBStateStatement := `select * from dbstate`
	err := tx.Unsafe().Get(&dbState, getDBStateStatement)
	if err != nil {
		log.Printf("We need to create a dbstate")
		//Suggest values on insert (triggers are going to override)
		dbState.SchemaVersion = SchemaVersion
		dbState.Identifier = fmt.Sprintf("%16x", time.Now().Unix())
		addMetaStatement, err := tx.Preparex(
			`insert dbstate set schemaversion = ?, identifier = ?, createddate = ?, modifieddate = ?`,
		)
		if err != nil {
			return dbState, err
		}
		//Create the record
		_, err = addMetaStatement.Exec(
			dbState.SchemaVersion,
			dbState.Identifier,
			dbState.CreateDate,
			dbState.ModifedDate,
		)
		if err != nil {
			log.Printf("Could not write out dbstate %v:%v", dbState, err)
			return dbState, err
		}
		//Returrn what is actually in dbState (it could override any of our suggestions)
		err = tx.Unsafe().Get(&dbState, getDBStateStatement)
		if err != nil {
			log.Printf("Could not get dbstate %v:%v", dbState, err)
			return dbState, err
		}
	}
	return dbState, nil
}

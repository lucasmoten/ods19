package dao

import (
	"log"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetUsers retrieves all users.
func (dao *DataAccessLayer) GetUsers() ([]models.ODUser, error) {
	tx := dao.MetadataDB.MustBegin()
	result, err := getUsersInTransaction(tx)
	if err != nil {
		log.Printf("Error in GetUsers: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return result, err
}

func getUsersInTransaction(tx *sqlx.Tx) ([]models.ODUser, error) {

	var result []models.ODUser
	getUsersStatement := `select distinguishedName, displayName, email from user`
	err := tx.Select(&result, getUsersStatement)
	if err != nil {
		log.Printf("Unable to execute query %s:%v", getUsersStatement, err)
	}
	return result, err
}

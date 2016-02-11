package dao

import (
	//"database/sql"
	//"fmt"
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
	"log"
)

// GetUsers retrieves the list of users.  There is no filtering by ACM yet.
func GetUsers(db *sqlx.DB) []models.ODUser {
	var result []models.ODUser
	getUsersStatement := `select * from user`
	err := db.Get(&result, getUsersStatement)
	if err != nil {
		log.Printf("Unable to execute query %s:%v", getUsersStatement, err)
	}
	return result
}

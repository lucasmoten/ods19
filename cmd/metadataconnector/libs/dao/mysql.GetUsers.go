package dao

import (
	//"database/sql"
	//"fmt"
	//"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
	"log"
)

// GetUsers retrieves the list of users.  There is no filtering by ACM yet.
func GetUsers(db *sqlx.DB) ([]string, error) {
	//XXX this is no good when the list is very large!!!!
	var result []string
	getUsersStatement := `select distinguishedName from user`
	err := db.Select(&result, getUsersStatement)
	if err != nil {
		log.Printf("Unable to execute query %s:%v", getUsersStatement, err)
	}
	return result, err
}

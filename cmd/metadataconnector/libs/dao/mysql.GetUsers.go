package dao

import (
	"log"

	"github.com/jmoiron/sqlx"
)

// GetUsers retrieves the list of users.  There is no filtering by ACM yet.
func GetUsers(db *sqlx.DB) ([]string, error) {
	// TODO this should return a User struct
	//XXX this is no good when the list is very large!!!!
	var result []string
	getUsersStatement := `select distinguishedName from user`
	err := db.Select(&result, getUsersStatement)
	if err != nil {
		log.Printf("Unable to execute query %s:%v", getUsersStatement, err)
	}
	return result, err
}

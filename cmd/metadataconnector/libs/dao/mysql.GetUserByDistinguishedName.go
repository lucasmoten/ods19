package dao

import (
	"database/sql"
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func GetUserByDistinguishedName(db *sqlx.DB, distinguishedName string) models.ODUser {
	var user models.ODUser
	getUserStatement := `select * from user where distinguishedName = ?`
	err := db.Get(&user, getUserStatement, distinguishedName)
	if err != nil {
		if err == sql.ErrNoRows {
			user.DistinguishedName = distinguishedName
			user.CreatedBy = distinguishedName
			createUser(db, &user)
		} // if err == sql.NoRows
	} // if err != nil
	return user
}

// createUser adds a new user definition to the database based upon the passed
// in ODUser object settings. At a minimm, createdBy and the distinguishedName
// of the user must already be assigned.  Once added, the record is retrieved
// and the user passed in by reference is updated with the remaining attributes
func createUser(db *sqlx.DB, user *models.ODUser) {
	addUserStatement, err := db.Prepare(`insert user set createdBy = ?, distinguishedName = ?, displayName = ?, email = ?`)
	if err != nil {
		panic(err)
	}

	result, err := addUserStatement.Exec(user.CreatedBy, user.DistinguishedName, "", "")
	if err != nil {
		panic(err)
	}
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		fmt.Println("No rows added from inserting user")
	}
	getUserStatement := `select * from user where distinguishedName = ?`
	err = db.Get(user, getUserStatement, user.DistinguishedName)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("User was not found even after just adding!")
		}
	}
}

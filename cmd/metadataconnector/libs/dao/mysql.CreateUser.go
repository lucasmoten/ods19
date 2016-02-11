package dao

import (
	"database/sql"
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// CreateUser adds the passed in user to the database. Once added, the record is
// retrieved and the user passed in by reference is updated with the remaining
// attributes
func CreateUser(db *sqlx.DB, user *models.ODUser) error {

	addUserStatement, err := db.Prepare(`insert user set createdBy = ?, distinguishedName = ?, displayName = ?, email = ?`)
	if err != nil {
		return err
	}

	result, err := addUserStatement.Exec(user.CreatedBy, user.DistinguishedName, user.DisplayName, user.Email)
	if err != nil {
		// Possible race condition here... Distinguished Name must be unique, and if
		// a parallel request is adding them then this attempt to insert will fail.
		// Attempt to retrieve them
		err := GetUserByDistinguishedName(db, user)
		if err != nil {
			return err
		}
		// Created already, and the get has populate the object, so return
		return nil
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowCount < 1 {
		fmt.Println("No rows added from inserting user")
	}
	// Get the newly added user
	err = GetUserByDistinguishedName(db, user)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("User was not found even after just adding!")
		}
		return err
	}
	return nil
}

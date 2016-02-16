package dao

import (
	"database/sql"
	"log"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// CreateUser adds the passed in user to the database. Once added, the record is
// retrieved and the user passed in by reference is updated with the remaining
// attributes
func CreateUser(db *sqlx.DB, user *models.ODUser) (*models.ODUser, error) {
	var dbUser *models.ODUser
	addUserStatement, err := db.Prepare(`insert user set createdBy = ?, distinguishedName = ?, displayName = ?, email = ?`)
	if err != nil {
		return dbUser, err
	}

	result, err := addUserStatement.Exec(user.CreatedBy, user.DistinguishedName, user.DisplayName, user.Email)
	if err != nil {
		// Possible race condition here... Distinguished Name must be unique, and if
		// a parallel request is adding them then this attempt to insert will fail.
		// Attempt to retrieve them
		dbUser, err = GetUserByDistinguishedName(db, user)
		if err != nil {
			return dbUser, err
		}
		// Created already, and the get has populated the object, so return
		return dbUser, nil
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		return dbUser, err
	}
	if rowCount < 1 {
		log.Printf("No rows were added when inserting the user!")
	}
	// Get the newly added user
	dbUser, err = GetUserByDistinguishedName(db, user)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User was not found even after just adding: %s", err.Error())
		} else {
			log.Printf("An error occurred retrieving newly added user: %s", err.Error())
		}
		return dbUser, err
	}
	return dbUser, nil
}
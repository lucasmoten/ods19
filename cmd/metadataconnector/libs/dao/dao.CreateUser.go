package dao

import (
	"database/sql"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
)

// CreateUser adds the passed in user to the database. Once added, the record is
// retrieved and the user passed in by reference is updated with the remaining
// attributes.
func (dao *DataAccessLayer) CreateUser(user models.ODUser) (models.ODUser, error) {
	tx := dao.MetadataDB.MustBegin()
	dbUser, err := createUserInTransaction(tx, user)
	if err != nil {
		log.Printf("Error in CreateUser: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbUser, err
}

func createUserInTransaction(tx *sqlx.Tx, user models.ODUser) (models.ODUser, error) {
	var dbUser models.ODUser
	addUserStatement, err := tx.Preparex(
		`insert user set createdBy = ?, distinguishedName = ?, displayName = ?, email = ?`)
	if err != nil {
		return dbUser, err
	}

	result, err := addUserStatement.Exec(user.CreatedBy, user.DistinguishedName, user.DisplayName, user.Email)
	if err != nil {
		// Possible race condition here... Distinguished Name must be unique, and if
		// a parallel request is adding them then this attempt to insert will fail.
		// Attempt to retrieve them
		dbUser, err = getUserByDistinguishedNameInTransaction(tx, user)
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
	dbUser, err = getUserByDistinguishedNameInTransaction(tx, user)
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

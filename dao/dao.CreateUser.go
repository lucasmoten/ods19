package dao

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// CreateUser adds the passed in user to the database. Once added, the record is
// retrieved and the user passed in by reference is updated with the remaining
// attributes.
func (dao *DataAccessLayer) CreateUser(user models.ODUser) (models.ODUser, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODUser{}, err
	}
	dbUser, err := createUserInTransaction(dao.GetLogger(), tx, user)
	if err != nil {
		dao.GetLogger().Error("Error in CreateUser", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbUser, err
}

func createUserInTransaction(logger zap.Logger, tx *sqlx.Tx, user models.ODUser) (models.ODUser, error) {
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
		logger.Warn("No rows were added when inserting the user!")
	}
	// Get the newly added user
	dbUser, err = getUserByDistinguishedNameInTransaction(tx, user)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Error("User was not found even after just adding", zap.String("err", err.Error()))
		} else {
			logger.Error("An error occurred retrieving newly added user", zap.String("err", err.Error()))
		}
		return dbUser, err
	}
	return dbUser, nil
}
package dao

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// CreateUser adds the passed in user to the database. Once added, the record is
// retrieved and the user passed in by reference is updated with the remaining
// attributes.
func (dao *DataAccessLayer) CreateUser(user models.ODUser) (models.ODUser, error) {
	defer util.Time("CreateUser")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODUser{}, err
	}
	dbUser, err := createUserInTransaction(tx, dao, user)
	if err != nil {
		dao.GetLogger().Error("Error in CreateUser", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbUser, err
}

func createUserInTransaction(tx *sqlx.Tx, dao *DataAccessLayer, user models.ODUser) (models.ODUser, error) {
	var dbUser models.ODUser
	addUserStatement, err := tx.Preparex(
		`insert user set createdBy = ?, distinguishedName = ?, displayName = ?, email = ?`)
	if err != nil {
		return dbUser, err
	}
	defer addUserStatement.Close()
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
		dao.GetLogger().Warn("no rows were added when inserting the user!")
	}
	// Get the newly added user
	dbUser, err = getUserByDistinguishedNameInTransaction(tx, user)
	if err != nil {
		if err == sql.ErrNoRows {
			dao.GetLogger().Error("user was not found even after just adding", zap.Error(err))
		} else {
			dao.GetLogger().Error("an error occurred retrieving newly added user", zap.Error(err))
		}
		return dbUser, err
	}
	return dbUser, nil
}

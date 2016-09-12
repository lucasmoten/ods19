package dao

import (
	"database/sql"

	configx "decipher.com/object-drive-server/configx"
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

func getAcmGranteeInTransaction(tx *sqlx.Tx, grantee string) (models.ODAcmGrantee, error) {
	var response models.ODAcmGrantee
	query := `
    select 
        grantee
        ,projectName
        ,projectDisplayName
        ,groupName
        ,userDistinguishedName
        ,displayName
    from acmgrantee  
    where
        grantee = ?`
	err := tx.Unsafe().Get(&response, query, grantee)
	if err != nil {
		return response, err
	}
	return response, err
}

func createAcmGranteeInTransaction(logger zap.Logger, tx *sqlx.Tx, acmGrantee models.ODAcmGrantee) (models.ODAcmGrantee, error) {

	// If grantee is for a user, check that user specified exists
	userDN := acmGrantee.UserDistinguishedName.String
	if acmGrantee.UserDistinguishedName.Valid && len(userDN) > 0 {
		userRequested := models.ODUser{}
		userRequested.DistinguishedName = userDN
		_, err := getUserByDistinguishedNameInTransaction(tx, userRequested)
		if err != nil && err == sql.ErrNoRows {
			// Not yet in database, we need to add them
			userRequested.DistinguishedName = userDN
			userRequested.DisplayName = models.ToNullString(configx.GetCommonName(userDN))
			userRequested.CreatedBy = userDN
			_, err = createUserInTransaction(logger, tx, userRequested)
		}
	}

	var dbAcmGrantee models.ODAcmGrantee
	addAcmGranteeStatement, err := tx.Preparex(
		`insert acmgrantee 
         set grantee = ?, projectName = ?, projectDisplayName = ?, groupName = ?, userDistinguishedName = ?, displayName = ?`)
	if err != nil {
		return dbAcmGrantee, err
	}

	result, err := addAcmGranteeStatement.Exec(acmGrantee.Grantee,
		acmGrantee.ProjectName, acmGrantee.ProjectDisplayName, acmGrantee.GroupName,
		acmGrantee.UserDistinguishedName, acmGrantee.DisplayName)
	if err != nil {
		// Possible race condition here... Grantee must be unique, and if
		// a parallel request is adding them then this attempt to insert will fail.
		// Attempt to retrieve them
		dbAcmGrantee, err = getAcmGranteeInTransaction(tx, acmGrantee.Grantee)
		if err != nil {
			return dbAcmGrantee, err
		}
		// Created already, and the get has populated the object, so return
		return dbAcmGrantee, nil
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		return dbAcmGrantee, err
	}
	if rowCount < 1 {
		logger.Warn("No rows were added when inserting the grantee!")
	}
	// Get the newly added grantee
	dbAcmGrantee, err = getAcmGranteeInTransaction(tx, acmGrantee.Grantee)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Error("Grantee was not found even after just adding", zap.String("err", err.Error()))
		} else {
			logger.Error("An error occurred retrieving newly added grantee", zap.String("err", err.Error()))
		}
		return dbAcmGrantee, err
	}
	return dbAcmGrantee, nil
}

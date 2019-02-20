package dao

import (
	"database/sql"
	"strings"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetAcmGrantee retrieves an existing AcmGrantee record by the grantee name.
func (dao *DataAccessLayer) GetAcmGrantee(grantee string) (models.ODAcmGrantee, error) {
	defer util.Time("GetAcmGrantee")()

	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODAcmGrantee{}, err
	}
	response, err := getAcmGranteeInTransaction(tx, grantee)
	if err != nil {
		dao.GetLogger().Error("error in getacmgrantee", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

// GetAcmGrantees retrieves a list of acm grantee records from a provided list of grantee names
func (dao *DataAccessLayer) GetAcmGrantees(grantees []string) ([]models.ODAcmGrantee, error) {
	defer util.Time("GetAcmGrantees")()

	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return []models.ODAcmGrantee{}, err
	}
	acmgrantees := []models.ODAcmGrantee{}
	var acmgrantee models.ODAcmGrantee
	for _, grantee := range grantees {
		acmgrantee, err = getAcmGranteeInTransaction(tx, grantee)
		if err == nil {
			acmgrantees = append(acmgrantees, acmgrantee)
			// we can't commit yet, because we are in a loop
		} else {
			if err != sql.ErrNoRows {
				dao.GetLogger().Error("error in getacmgrantees", zap.Error(err))
				tx.Rollback()
				return acmgrantees, err
			}
			err = nil
		}
	}
	tx.Commit()
	return acmgrantees, err
}

func getAcmGranteeInTransaction(tx *sqlx.Tx, grantee string) (models.ODAcmGrantee, error) {
	var response models.ODAcmGrantee
	query := `
    select 
        grantee
		,resourceString
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

// CreateAcmGrantee creates an AcmGrantee record if it does not already exist, otherwise fetches by the grantee name.
func (dao *DataAccessLayer) CreateAcmGrantee(acmGrantee models.ODAcmGrantee) (models.ODAcmGrantee, error) {
	defer util.Time("CreateAcmGrantee")()
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	retryOnErrorMessageContains := []string{"Duplicate entry", "Deadlock", "Lock wait timeout exceeded", sql.ErrNoRows.Error()}
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODAcmGrantee{}, err
	}
	response, err := createAcmGranteeInTransaction(tx, dao, acmGrantee)
	for retryCounter > 0 && err != nil && containsAny(err.Error(), retryOnErrorMessageContains) {
		dao.GetLogger().Debug("restarting transaction for CreateAcmGrantee", zap.String("retryReason", firstMatch(err.Error(), retryOnErrorMessageContains)), zap.Int64("retryCounter", retryCounter))
		tx.Rollback()
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		retryCounter--
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			dao.GetLogger().Error("could not begin transaction", zap.Error(err))
			return models.ODAcmGrantee{}, err
		}
		response, err = createAcmGranteeInTransaction(tx, dao, acmGrantee)
	}
	if err != nil {
		dao.GetLogger().Error("error in createacmgrantee", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func createAcmGranteeInTransaction(tx *sqlx.Tx, dao *DataAccessLayer, acmGrantee models.ODAcmGrantee) (models.ODAcmGrantee, error) {

	// If grantee is for a user, check that user specified exists
	userDN := acmGrantee.UserDistinguishedName.String
	if acmGrantee.UserDistinguishedName.Valid && len(userDN) > 0 {
		userRequested := models.ODUser{}
		userRequested.DistinguishedName = userDN
		_, err := getUserByDistinguishedNameInTransaction(tx, userRequested)
		if err != nil && err == sql.ErrNoRows {
			// Not yet in database, we need to add them
			userRequested.DistinguishedName = userDN
			userRequested.DisplayName = models.ToNullString(config.GetCommonName(userDN))
			userRequested.CreatedBy = userDN
			_, err = createUserInTransaction(tx, dao, userRequested)
		}
		if !acmGrantee.DisplayName.Valid || acmGrantee.DisplayName.String == "" {
			acmGrantee.DisplayName = models.ToNullString(config.GetCommonName(userDN))
		}
	} else if acmGrantee.GroupName.Valid {
		projectDisplayName := acmGrantee.ProjectDisplayName.String
		groupName := acmGrantee.GroupName.String
		if !acmGrantee.DisplayName.Valid || acmGrantee.DisplayName.String == "" {
			acmGrantee.DisplayName = models.ToNullString(strings.TrimSpace(projectDisplayName + " " + groupName))
		}
	}
	if !acmGrantee.ResourceString.Valid || acmGrantee.ResourceString.String == "" {
		acmGrantee.ResourceString = models.ToNullString(acmGrantee.ResourceName())
	}
	acmGrantee.ResourceString = models.ToNullString(removeDisplayNameFromResourceString(acmGrantee.ResourceString.String))
	acmGrantee.Grantee = models.AACFlatten(acmGrantee.Grantee)

	var dbAcmGrantee models.ODAcmGrantee
	dbAcmGrantee, err := getAcmGranteeInTransaction(tx, acmGrantee.Grantee)
	if err != nil || dbAcmGrantee.Grantee != acmGrantee.Grantee {

		addAcmGranteeStatement, err := tx.Preparex(
			`insert acmgrantee 
         set grantee = ?, resourceString = ?, projectName = ?, projectDisplayName = ?, groupName = ?, userDistinguishedName = ?, displayName = ?`)
		if err != nil {
			return dbAcmGrantee, err
		}
		defer addAcmGranteeStatement.Close()
		result, err := addAcmGranteeStatement.Exec(acmGrantee.Grantee, acmGrantee.ResourceString,
			acmGrantee.ProjectName, acmGrantee.ProjectDisplayName, acmGrantee.GroupName,
			acmGrantee.UserDistinguishedName, acmGrantee.DisplayName)
		if err != nil {
			dao.GetLogger().Warn("error executing addacmgranteestatement", zap.Error(err))
			// Possible race condition here... Grantee must be unique, and if
			// a parallel request is adding them then this attempt to insert will fail.
			// Attempt to retrieve them
			dbAcmGrantee, err = getAcmGranteeInTransaction(tx, acmGrantee.Grantee)
			if err != nil {
				dao.GetLogger().Warn("error getting acmgrantee in transaction", zap.Error(err))
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
			dao.GetLogger().Warn("no rows were added when inserting the grantee!")
		}
	}
	//addAcmGranteeStatement.Close()
	// Get the newly added grantee
	dbAcmGrantee, err = getAcmGranteeInTransaction(tx, acmGrantee.Grantee)
	if err != nil {
		if err == sql.ErrNoRows {
			dao.GetLogger().Error("grantee was not found even after just adding", zap.Error(err))
		} else {
			dao.GetLogger().Error("an error occurred retrieving newly added grantee", zap.Error(err))
		}
		return dbAcmGrantee, err
	}
	return dbAcmGrantee, nil
}

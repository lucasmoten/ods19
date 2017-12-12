package dao

import (
	"database/sql"
	"strings"
	"time"

	"github.com/deciphernow/object-drive-server/config"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetAcmGrantee retrieves an exsiting AcmGrantee record by the grantee name.
func (dao *DataAccessLayer) GetAcmGrantee(grantee string) (models.ODAcmGrantee, error) {
	defer util.Time("GetAcmGrantee")()

	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.String("err", err.Error()))
		return models.ODAcmGrantee{}, err
	}
	response, err := getAcmGranteeInTransaction(tx, grantee)
	if err != nil {
		dao.GetLogger().Error("error in getacmgrantee", zap.String("err", err.Error()))
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
		dao.GetLogger().Error("could not begin transaction", zap.String("err", err.Error()))
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
				dao.GetLogger().Error("error in getacmgrantees", zap.String("err", err.Error()))
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
	logger := dao.GetLogger()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		logger.Error("could not begin transaction", zap.String("err", err.Error()))
		return models.ODAcmGrantee{}, err
	}
	deadlockRetryCounter := dao.DeadlockRetryCounter
	deadlockRetryDelay := dao.DeadlockRetryDelay
	deadlockMessage := "Deadlock"
	response, err := createAcmGranteeInTransaction(dao.GetLogger(), tx, acmGrantee)
	for deadlockRetryCounter > 0 && err != nil && strings.Contains(err.Error(), deadlockMessage) {
		logger.Info("deadlock in createacmgrantee, restarting transaction", zap.Int64("deadlockRetryCounter", deadlockRetryCounter))
		time.Sleep(time.Duration(deadlockRetryDelay) * time.Millisecond)
		// Cancel the old transaction and start a new one
		tx.Rollback()
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction", zap.String("err", err.Error()))
			return models.ODAcmGrantee{}, err
		}
		// Retry the create
		deadlockRetryCounter--
		response, err = createAcmGranteeInTransaction(dao.GetLogger(), tx, acmGrantee)
	}
	if err != nil {
		logger.Error("error in createacmgrantee", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func createAcmGranteeInTransaction(logger *zap.Logger, tx *sqlx.Tx, acmGrantee models.ODAcmGrantee) (models.ODAcmGrantee, error) {

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
			_, err = createUserInTransaction(logger, tx, userRequested)
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
	addAcmGranteeStatement, err := tx.Preparex(
		`insert acmgrantee 
         set grantee = ?, resourceString = ?, projectName = ?, projectDisplayName = ?, groupName = ?, userDistinguishedName = ?, displayName = ?`)
	if err != nil {
		return dbAcmGrantee, err
	}

	result, err := addAcmGranteeStatement.Exec(acmGrantee.Grantee, acmGrantee.ResourceString,
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
		logger.Warn("no rows were added when inserting the grantee!")
	}
	// Get the newly added grantee
	dbAcmGrantee, err = getAcmGranteeInTransaction(tx, acmGrantee.Grantee)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Error("grantee was not found even after just adding", zap.String("err", err.Error()))
		} else {
			logger.Error("an error occurred retrieving newly added grantee", zap.String("err", err.Error()))
		}
		return dbAcmGrantee, err
	}
	return dbAcmGrantee, nil
}

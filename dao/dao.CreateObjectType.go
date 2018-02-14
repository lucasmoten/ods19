package dao

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
)

// CreateObjectType adds a new object type definition to the database based upon
// the passed in object type settings.  At a minimm, createdBy and the name of
// the object type must exist.  Once added, the record is retrieved and the
// object type passed in by reference is updated with the remaining attributes
func (dao *DataAccessLayer) CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error) {
	defer util.Time("CreateObjectType")()
	logger := dao.GetLogger()
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	retryOnErrorMessageContains := []string{"Duplicate entry", "Deadlock", "Lock wait timeout exceeded"}
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		logger.Error("could not begin transaction", zap.Error(err))
		return models.ODObjectType{}, err
	}
	dbObjectType, err := createObjectTypeInTransaction(tx, objectType)
	for retryCounter > 0 && err != nil && containsAny(err.Error(), retryOnErrorMessageContains) {
		logger.Debug("restarting transaction for createObjectTypeInTransaction", zap.String("retryReason", firstMatch(err.Error(), retryOnErrorMessageContains)), zap.Int64("retryCounter", retryCounter))
		tx.Rollback()
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		retryCounter--
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction", zap.Error(err))
			return models.ODObjectType{}, err
		}
		dbObjectType, err = createObjectTypeInTransaction(tx, objectType)
	}
	if err != nil {
		logger.Error("error in CreateObjectType", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectType, err
}

func createObjectTypeInTransaction(tx *sqlx.Tx, objectType *models.ODObjectType) (models.ODObjectType, error) {
	var dbObjectType models.ODObjectType
	addObjectTypeStatement, err := tx.Preparex(`
		insert ignore into object_type set createdBy = ?, name = ?, description = ?, contentConnector = ?`)
	if err != nil {
		return dbObjectType, fmt.Errorf("CreateObjectType error preparing add object type statement, %s", err.Error())
	}
	defer addObjectTypeStatement.Close()
	// Add it
	if _, err := addObjectTypeStatement.Exec(objectType.CreatedBy, objectType.Name,
		objectType.Description.String, objectType.ContentConnector.String); err != nil {
		return dbObjectType, err
	}
	// Retrieve it
	getObjectTypeStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector
    from object_type 
    where 
        name = ? 
        and isdeleted = 0 
    order by createdDate desc limit 1`
	err = tx.Get(&dbObjectType, getObjectTypeStatement, objectType.Name)
	return dbObjectType, err
}

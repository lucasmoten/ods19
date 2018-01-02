package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"encoding/hex"

	"github.com/deciphernow/object-drive-server/config"
	"github.com/deciphernow/object-drive-server/crypto"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// CreateObject ...
func (dao *DataAccessLayer) CreateObject(object *models.ODObject) (models.ODObject, error) {
	defer util.Time("CreateObject")()
	logger := dao.GetLogger()
	tx, err := dao.MetadataDB.Beginx()
	var obj models.ODObject
	if err != nil {
		logger.Error("could not begin transaction", zap.Error(err))
		return models.ODObject{}, err
	}
	var dbObject models.ODObject
	var acmCreated bool

	deadlockRetryCounter := dao.DeadlockRetryCounter
	deadlockRetryDelay := dao.DeadlockRetryDelay
	deadlockMessage := "Deadlock"
	dbObject, acmCreated, err = createObjectInTransaction(logger, tx, dao, object)
	// Deadlock trapper on acm
	for deadlockRetryCounter > 0 && err != nil && strings.Contains(err.Error(), deadlockMessage) {
		logger.Info("deadlock in CreateObject, restarting transaction", zap.Int64("deadlockRetryCounter", deadlockRetryCounter))
		time.Sleep(time.Duration(deadlockRetryDelay) * time.Millisecond)
		// Cancel the old transaction and start a new one
		tx.Rollback()
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction", zap.Error(err))
			return models.ODObject{}, err
		}
		// Retry the create
		deadlockRetryCounter--
		dbObject, acmCreated, err = createObjectInTransaction(logger, tx, dao, object)
	}
	if err != nil {
		logger.Error("error in CreateObject", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
		// Calculate in background and as separate transaction...
		if acmCreated {
			runasync := true
			if runasync {
				if err := insertAssociationOfACMToModifiedByIfValid(dao, dbObject); err != nil {
					logger.Error("error associating the acm on this object to the user that created it!", zap.Error(err), zap.String("ObjectID", hex.EncodeToString(dbObject.ID)), zap.String("modifiedby", dbObject.ModifiedBy), zap.Int64("acmID", dbObject.ACMID))
				}
				go func() {
					done := make(chan bool)
					timeout := time.After(60 * time.Second)
					go dao.AssociateUsersToNewACM(dbObject, done)

					for {
						select {
						case <-timeout:
							dao.GetLogger().Warn("createobject call to associateuserstonewacm timed out")
							return
						case <-done:
							return
						}
					}
				}()
			} else {
				done := make(chan bool, 1)
				dao.AssociateUsersToNewACM(dbObject, done)
			}
		}
		// Refetch
		obj, err = dao.GetObject(dbObject, true)
		if err != nil {
			logger.Error("error in CreateObject subsequent GetObject call]")
			return models.ODObject{}, err
		}
	}
	return obj, err
}

func createObjectInTransaction(logger *zap.Logger, tx *sqlx.Tx, dao *DataAccessLayer, object *models.ODObject) (models.ODObject, bool, error) {

	var dbObject models.ODObject
	var acmCreated bool

	if len(object.TypeID) == 0 {
		object.TypeID = nil
	}
	if len(object.ParentID) == 0 {
		object.ParentID = nil
	}
	if object.CreatedBy == "" {
		return dbObject, acmCreated, errors.New("cannot create object, createdby field is missing")
	}

	// Add creator if not yet present. (From direct DAO calls)
	userRequested := models.ODUser{}
	userRequested.DistinguishedName = object.CreatedBy
	_, err := getUserByDistinguishedNameInTransaction(tx, userRequested)
	if err != nil && err == sql.ErrNoRows {
		// Not yet in database, we need to add them
		userRequested.DistinguishedName = object.CreatedBy
		userRequested.DisplayName = models.ToNullString(config.GetCommonName(object.CreatedBy))
		userRequested.CreatedBy = object.CreatedBy
		userCreated := models.ODUser{}
		userCreated, err = createUserInTransaction(logger, tx, userRequested)
		object.CreatedBy = userCreated.DistinguishedName
	}

	if object.TypeID == nil {
		objectType, err := getObjectTypeByNameInTransaction(tx, object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			return dbObject, acmCreated, fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	if len(object.Name) == 0 {
		object.Name = "New " + object.TypeName.String
	}

	// Assign a random content connector value if this object doesnt have one
	//
	// This value is normally set for objects having a content stream.  For folder objects, this value is not
	// initially set, but we leverage it below as a psuedo unique identifier for fetching the created
	// record. This is because the truly unique identifier is generated in the database as a non-predictive
	// GUID.
	if len(object.ContentConnector.String) == 0 {
		object.ContentConnector = models.ToNullString(crypto.CreateRandomName())
	}

	// Normalize ACM
	newACMNormalized, err := normalizedACM(object.RawAcm.String)
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("Error normalizing ACM on new object: %v (acm: %s)", err.Error(), object.RawAcm.String)
	}
	object.RawAcm.String = newACMNormalized
	acmCreated, err = setObjectACM2ForObjectInTransaction(tx, dao, object)
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("Error assigning ACM ID for object: %s", err.Error())
	}

	addObjectStatement, err := tx.Preparex(`insert object set 
        createdBy = ?
        ,typeId = ?
        ,name = ?
        ,description = ?
        ,parentId = ?
        ,contentConnector = ?
        ,rawAcm = ?
        ,contentType = ?
        ,contentSize = ?
        ,contentHash = ?
        ,encryptIV = ?
        ,containsUSPersonsData = ?
        ,exemptFromFOIA = ?
        ,ownedBy = ?
        ,acmId = ?
    `)
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("CreateObject Preparing add object statement, %s", err.Error())
	}
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String,
		object.ContentType.String, object.ContentSize.Int64, object.ContentHash,
		object.EncryptIV, object.ContainsUSPersonsData, object.ExemptFromFOIA, object.OwnedBy.String,
		object.ACMID)
	if err != nil {
		errMsg := err.Error()
		return dbObject, acmCreated, fmt.Errorf("createobject error executing add object statement, %s", errMsg)
	}
	err = addObjectStatement.Close()
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("createobject error closing addobjectstatement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("createobject error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return dbObject, acmCreated, fmt.Errorf("createobject object inserted but no rows affected")
	}

	// Get the populated fields (id, createddate, modifieddate, etc) of the newly created object and assign to returned object.
	// This assumes most recent created by the user of the type and name.
	getObjectStatement := `
    select 
        o.id    
        ,o.createdDate
        ,o.createdBy
        ,o.modifiedDate
        ,o.modifiedBy
        ,o.isDeleted
        ,o.deletedDate
        ,o.deletedBy
        ,o.isAncestorDeleted
        ,o.isExpunged
        ,o.expungedDate
        ,o.expungedBy
        ,o.changeCount
        ,o.changeToken
        ,o.ownedBy
        ,o.typeId
        ,o.name
        ,o.description
        ,o.parentId
        ,o.contentConnector
        ,o.rawAcm
        ,o.contentType
        ,o.contentSize
        ,o.contentHash
        ,o.encryptIV
        ,o.containsUSPersonsData
        ,o.exemptFromFOIA
        ,ot.name typeName
		,o.acmid   
    from object o 
        inner join object_type ot on o.typeId = ot.id 
    where 
        o.createdby = ? 
        and o.typeId = ? 
        and o.name = ? 
        and o.contentConnector = ?
        and o.isdeleted = 0 
    order by o.createddate desc limit 1`
	err = tx.Get(&dbObject, getObjectStatement, object.CreatedBy, object.TypeID, object.Name, object.ContentConnector)
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("CreateObject Error retrieving object, %s", err.Error())
	}

	// Add properties of object.Properties []models.ODObjectPropertyEx
	for i, property := range object.Properties {
		if property.Name != "" {
			var objectProperty models.ODProperty
			objectProperty.CreatedBy = dbObject.CreatedBy
			objectProperty.Name = property.Name
			if property.Value.Valid {
				objectProperty.Value.String = property.Value.String
				objectProperty.Value.Valid = true
			}
			if property.ClassificationPM.Valid {
				objectProperty.ClassificationPM.String = property.ClassificationPM.String
				objectProperty.ClassificationPM.Valid = true
			}
			dbProperty, err := addPropertyToObjectInTransaction(tx, dbObject, &objectProperty)
			if err != nil {
				return dbObject, acmCreated, fmt.Errorf("Error saving property %d (%s) when creating object", i, property.Name)
			}
			if dbProperty.ID == nil {
				return dbObject, acmCreated, fmt.Errorf("New property does not have an ID")
			}
		}
	}

	// Add permissions
	for i, permission := range object.Permissions {
		if !permission.IsDeleted && permission.Grantee != "" {
			permission.CreatedBy = dbObject.CreatedBy
			dbPermission, err := addPermissionToObjectInTransaction(logger, tx, dbObject, &permission)
			if err != nil {
				return dbObject, acmCreated, fmt.Errorf("Error saving permission # %d {Grantee: \"%s\") when creating object:%v", i, permission.Grantee, err)
			}
			if dbPermission.ModifiedBy != permission.CreatedBy {
				return dbObject, acmCreated, fmt.Errorf("When creating object, permission did not get modifiedby set to createdby")
			}
			object.Permissions[i] = dbPermission
		}
	}

	return dbObject, acmCreated, nil
}

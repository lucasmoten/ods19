package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"encoding/hex"

	"bitbucket.di2e.net/dime/object-drive-server/ciphertext"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// CreateObject ...
func (dao *DataAccessLayer) CreateObject(object *models.ODObject) (models.ODObject, error) {
	defer util.Time("CreateObject")()
	var obj models.ODObject
	dao.GetLogger().Debug("dao starting txn for CreateObject")
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODObject{}, err
	}
	var dbObject models.ODObject
	var acmCreated bool

	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	retryOnErrorMessageContains := []string{"Duplicate entry", "Deadlock", "Lock wait timeout exceeded", "Field name must be unique"}
	dao.GetLogger().Debug("dao passing  txn into createObjectInTransaction")
	dbObject, acmCreated, err = createObjectInTransaction(tx, dao, object)
	dao.GetLogger().Debug("dao returned txn from createObjectInTransaction")
	dao.GetLogger().Debug("dao checking response from creating object")
	for retryCounter > 0 && err != nil && util.ContainsAny(err.Error(), retryOnErrorMessageContains) {
		dao.GetLogger().Debug("dao restarting transaction for creating object", zap.String("retryReason", util.FirstMatch(err.Error(), retryOnErrorMessageContains)), zap.Int64("retryCounter", retryCounter))
		dao.GetLogger().Debug("-- txn rollback", zap.Int64("retryCounter", retryCounter))
		tx.Rollback()
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		retryCounter--
		dao.GetLogger().Debug("-- txn begin", zap.Int64("retryCounter", retryCounter))
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			dao.GetLogger().Error("could not begin transaction", zap.Error(err))
			return models.ODObject{}, err
		}
		dao.GetLogger().Debug("dao passing  txn into createObjectInTransaction during retry")
		dbObject, acmCreated, err = createObjectInTransaction(tx, dao, object)
		dao.GetLogger().Debug("dao returned txn from createObjectInTransaction during retry")
	}
	if err != nil {
		dao.GetLogger().Error("error in CreateObject", zap.Error(err))
		dao.GetLogger().Debug("dao rolling back txn for CreateObject")
		tx.Rollback()
	} else {
		dao.GetLogger().Debug("dao committed transaction for CreateObject")
		dao.GetLogger().Debug("dao committing txn for CreateObject")
		tx.Commit()
	}
	dao.GetLogger().Debug("dao finished txn for CreateObject")
	if err == nil {
		dao.GetLogger().Debug("dao checking if new acm created")
		// Calculate in background and as separate transaction...
		if acmCreated {
			runasync := true
			if runasync {
				dao.GetLogger().Debug("dao determined new acm was created, and will associate asynchronously")
				if err := insertAssociationOfACMToModifiedByIfValid(dao, dbObject); err != nil {
					dao.GetLogger().Error("error associating the acm on this object to the user that created it!", zap.Error(err), zap.String("ObjectID", hex.EncodeToString(dbObject.ID)), zap.String("modifiedby", dbObject.ModifiedBy), zap.Int64("acmID", dbObject.ACMID))
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
				dao.GetLogger().Debug("dao determined new acm was created, and will associate now")
				done := make(chan bool, 1)
				dao.AssociateUsersToNewACM(dbObject, done)
			}
		} else {
			dao.GetLogger().Debug("dao acm was pre-existing")
		}
		// Refetch
		dao.GetLogger().Debug("dao retrieving object that was just created")
		obj, err = dao.GetObject(dbObject, true)
		if err != nil {
			dao.GetLogger().Error("error in CreateObject subsequent GetObject call")
			return models.ODObject{}, err
		}
		dao.GetLogger().Debug("dao object retrieved, ready to return")
	}
	return obj, err
}

func createObjectInTransaction(tx *sqlx.Tx, dao *DataAccessLayer, object *models.ODObject) (models.ODObject, bool, error) {

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
	if object.TypeID == nil {
		return dbObject, acmCreated, errors.New("cannot create object, typeid field is missing")
	}

	// Add creator if not yet present. (From direct DAO calls)
	userRequested := models.ODUser{}
	userRequested.DistinguishedName = object.CreatedBy
	dao.GetLogger().Debug("dao passing  txn into getUserByDistinguishedNameInTransaction")
	_, err := getUserByDistinguishedNameInTransaction(tx, userRequested)
	dao.GetLogger().Debug("dao returned txn from getUserByDistinguishedNameInTransaction")
	if err != nil && err == sql.ErrNoRows {
		// Not yet in database, we need to add them
		userRequested.DistinguishedName = object.CreatedBy
		userRequested.DisplayName = models.ToNullString(config.GetCommonName(object.CreatedBy))
		userRequested.CreatedBy = object.CreatedBy
		userCreated := models.ODUser{}
		dao.GetLogger().Debug("dao passing  txn into createUserInTransaction")
		userCreated, err = createUserInTransaction(tx, dao, userRequested)
		dao.GetLogger().Debug("dao returned txn from createUserInTransaction")
		object.CreatedBy = userCreated.DistinguishedName
	}

	if len(object.Name) == 0 {
		object.Name = "New " + object.TypeName.String
	}

	// Assign a random content connector value if this object doesn't have one
	//
	// This value is normally set for objects having a content stream.  For folder objects, this value is not
	// initially set, but we leverage it below as a pseudo unique identifier for fetching the created
	// record. This is because the truly unique identifier is generated in the database as a non-predictive
	// GUID.
	if len(object.ContentConnector.String) == 0 {
		object.ContentConnector = models.ToNullString(ciphertext.CreateRandomName())
	}

	// Normalize ACM
	newACMNormalized, err := normalizedACM(object.RawAcm.String)
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("Error normalizing ACM on new object: %v (acm: %s)", err.Error(), object.RawAcm.String)
	}
	object.RawAcm.String = newACMNormalized
	dao.GetLogger().Debug("dao passing  txn into setObjectACM2ForObjectInTransaction")
	acmCreated, err = setObjectACM2ForObjectInTransaction(tx, dao, object)
	dao.GetLogger().Debug("dao returned txn from setObjectACM2ForObjectInTransaction")
	if err != nil {
		return dbObject, acmCreated, fmt.Errorf("Error assigning ACM ID for object: %s", err.Error())
	}
	dao.GetLogger().Debug("dao preparing stmt for insert to object")
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
	dao.GetLogger().Debug("dao prepared stmt will have deferred close")
	defer addObjectStatement.Close()
	dao.GetLogger().Debug("dao executing stmt for insert to object")
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.RawAcm.String,
		object.ContentType.String, object.ContentSize.Int64, object.ContentHash,
		object.EncryptIV, object.ContainsUSPersonsData, object.ExemptFromFOIA, object.OwnedBy.String,
		object.ACMID)
	dao.GetLogger().Debug("dao checking response from insert to object")
	if err != nil {
		errMsg := err.Error()
		return dbObject, acmCreated, fmt.Errorf("createobject error executing add object statement, %s", errMsg)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		dao.GetLogger().Error("dao error getting rows affected in createObjectInTransaction")
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
	dao.GetLogger().Debug("dao txn used to get inserted object")
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
			dao.GetLogger().Debug("dao passing  txn into addPropertyToObjectInTransaction")
			dbProperty, err := addPropertyToObjectInTransaction(tx, dbObject, &objectProperty)
			dao.GetLogger().Debug("dao returned txn from addPropertyToObjectInTransaction")
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
			dao.GetLogger().Debug("dao passing  txn into addPermissionToObjectInTransaction")
			dbPermission, err := addPermissionToObjectInTransaction(tx, dao, dbObject, &permission)
			dao.GetLogger().Debug("dao returned txn from addPermissionToObjectInTransaction")
			if err != nil {
				return dbObject, acmCreated, fmt.Errorf("Error saving permission # %d {Grantee: \"%s\") when creating object:%v", i, permission.Grantee, err)
			}
			if dbPermission.ModifiedBy != permission.CreatedBy {
				return dbObject, acmCreated, fmt.Errorf("When creating object, permission did not get modifiedby set to createdby")
			}
			object.Permissions[i] = dbPermission
		}
	}
	dao.GetLogger().Debug("dao completed complex nested txn series for createObjectInTransaction")
	return dbObject, acmCreated, nil
}

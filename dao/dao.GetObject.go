package dao

import (
	"database/sql"
	"encoding/hex"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetObject uses the passed in object and makes the appropriate sql calls to
// the database to retrieve and return the requested object by ID. Optionally,
// loadProperties flag pulls in nested properties associated with the object.
func (dao *DataAccessLayer) GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error) {
	defer util.Time("GetObject")()
	loadPermissions := true
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODObject{}, err
	}
	dbObject, err := getObjectInTransaction(dao, tx, object, loadPermissions, loadProperties)
	if err != nil {
		if err != sql.ErrNoRows {
			dao.GetLogger().Error("error in getobject", zap.Error(err))
		} else {
			dao.GetLogger().Info("getobject requested id not found", zap.String("id", hex.EncodeToString(object.ID)))
		}
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObject, err
}

func getObjectInTransaction(dao *DataAccessLayer, tx *sqlx.Tx, object models.ODObject, loadPermissions, loadProperties bool) (models.ODObject, error) {
	var dbObject models.ODObject

	if len(object.ID) == 0 {
		return dbObject, ErrMissingID
	}

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
        ,o.acmId acmid
    from object o 
        inner join object_type ot on o.typeid = ot.id 
	where o.id = ?`
	err := tx.Get(&dbObject, getObjectStatement, object.ID)
	if err != nil {
		return dbObject, err
	}

	// Load Permissions
	if loadPermissions {
		dbPermissions, dbPermErr := getPermissionsForObjectInTransaction(dao, tx, object)
		dbObject.Permissions = dbPermissions
		if dbPermErr != nil {
			err = dbPermErr
			return dbObject, err
		}
	}

	// Load properties if requested
	if loadProperties {
		dbProperties, dbPropErr := getPropertiesForObjectInTransaction(tx, object)
		dbObject.Properties = dbProperties
		if dbPropErr != nil {
			err = dbPropErr
			return dbObject, err
		}
	}

	// Done
	return dbObject, nil
}

package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetObject uses the passed in object and makes the appropriate sql calls to
// the database to retrieve and return the requested object by ID. Optionally,
// loadProperties flag pulls in nested properties associated with the object.
func (dao *DataAccessLayer) GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObject{}, err
	}
	dbObject, err := getObjectInTransaction(tx, object, loadProperties)
	if err != nil {
		dao.GetLogger().Error("Error in GetObject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObject, err
}

func getObjectInTransaction(tx *sqlx.Tx, object models.ODObject, loadProperties bool) (models.ODObject, error) {
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
        ,o.ownedByNew
        ,o.isPDFAvailable
        ,o.isStreamStored
        ,o.containsUSPersonsData
        ,o.exemptFromFOIA
        ,ot.name typeName     
    from object o 
        inner join object_type ot on o.typeid = ot.id 
    where o.id = ?`
	err := tx.Unsafe().Get(&dbObject, getObjectStatement, object.ID)
	if err != nil {
		return dbObject, err
	}

	// Load Permissions
	dbPermissions, dbPermErr := getPermissionsForObjectInTransaction(tx, object)
	dbObject.Permissions = dbPermissions
	if dbPermErr != nil {
		err = dbPermErr
		return dbObject, err
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

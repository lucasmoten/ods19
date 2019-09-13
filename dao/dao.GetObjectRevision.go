package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetObjectRevision uses the passed in object and makes the appropriate sql
// calls to the database to retrieve and return the requested object by ID and
// changeCount. Optionally, loadProperties flag pulls in nested properties
// associated with this revision of the object.
func (dao *DataAccessLayer) GetObjectRevision(object models.ODObject, loadProperties bool) (models.ODObject, error) {
	defer util.Time("GetObjectRevision")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObject{}, err
	}
	dbObject, err := getObjectRevisionInTransaction(dao, tx, object, loadProperties)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectRevision", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObject, err
}

func getObjectRevisionInTransaction(dao *DataAccessLayer, tx *sqlx.Tx, object models.ODObject, loadProperties bool) (models.ODObject, error) {
	var dbObject models.ODObject

	query := `
    select 
        ao.id
        ,ao.createdDate
        ,ao.createdBy
        ,ao.modifiedDate
        ,ao.modifiedBy
        ,(ao.isDeleted | o.isDeleted) isDeleted
        ,ao.deletedDate
        ,ao.deletedBy
        ,(ao.isAncestorDeleted | o.isAncestorDeleted) isAncestorDeleted
        ,(ao.isExpunged | o.isExpunged) isExpunged
        ,ao.expungedDate
        ,ao.expungedBy
        ,ao.changeCount
        ,ao.changeToken
        ,ao.ownedBy
        ,ao.typeId
        ,ao.name
        ,ao.description
        ,ao.parentId
        ,ao.contentConnector
        ,ao.rawAcm
        ,ao.contentType
        ,ao.contentSize
        ,ao.contentHash
        ,ao.encryptIV
        ,ao.containsUSPersonsData
        ,ao.exemptFromFOIA
        ,ot.name typeName
        ,ao.acmid
    from a_object ao 
        inner join object o on ao.id = o.id
        inner join object_type ot on ao.typeid = ot.id
    where 
        o.isexpunged = 0
        and ao.isexpunged = 0 
        and ao.id = ? 
        and ao.changeCount = ?
            `
	err := tx.Unsafe().Get(&dbObject, query, object.ID, object.ChangeCount)
	if err == nil {
		dbPermissions, dbPermErr := getPermissionsForObjectInTransaction(dao, tx, object)
		dbObject.Permissions = dbPermissions
		if dbPermErr != nil {
			err = dbPermErr
		} else {
			// Load properties if requested
			if loadProperties {
				dbProperties, dbPropErr := getPropertiesForObjectRevisionInTransaction(tx, object)
				dbObject.Properties = dbProperties
				if dbPropErr != nil {
					err = dbPropErr
				}
			}
		}
	}
	return dbObject, err
}

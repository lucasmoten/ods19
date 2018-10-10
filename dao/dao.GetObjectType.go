package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetObjectType uses the passed in objectType and makes the appropriate sql
// calls to the database to retrieve and return the requested object type by ID.
func (dao *DataAccessLayer) GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error) {
	defer util.Time("GetObjectType")()
	dao.GetLogger().Debug("dao starting txn for GetObjectType")
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return nil, err
	}
	dao.GetLogger().Debug("dao passing  txn into getObjectTypeInTransaction")
	dbObjectType, err := getObjectTypeInTransaction(tx, objectType)
	dao.GetLogger().Debug("dao returned txn from getObjectTypeInTransaction")
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectType", zap.Error(err))
		dao.GetLogger().Debug("dao rolling back txn for GetObjectType")
		tx.Rollback()
	} else {
		dao.GetLogger().Debug("dao committing txn for GetObjectType")
		tx.Commit()
	}
	dao.GetLogger().Debug("dao finished txn for GetObjectType")
	return dbObjectType, err
}

func getObjectTypeInTransaction(tx *sqlx.Tx, objectType models.ODObjectType) (*models.ODObjectType, error) {
	var dbObjectType models.ODObjectType
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
    from
        object_type
    where
        id = ?    
    `
	err := tx.Get(&dbObjectType, getObjectTypeStatement, objectType.ID)
	if err != nil {
		return &dbObjectType, err
	}

	return &dbObjectType, err
}

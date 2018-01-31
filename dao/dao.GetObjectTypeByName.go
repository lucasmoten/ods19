package dao

import (
	"database/sql"

	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
)

// GetObjectTypeByName looks up an object type by its name, and if it doesn't
// exist, optionally calls CreateObjectType to add it.
func (dao *DataAccessLayer) GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {
	defer util.Time("GetObjectTypeByName")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectType{}, err
	}
	objectType, err := getObjectTypeByNameInTransaction(tx, typeName)
	if err != nil {
		tx.Rollback()
		if (err == sql.ErrNoRows) && addIfMissing {
			objectType, err = dao.CreateObjectType(&models.ODObjectType{Name: typeName, CreatedBy: createdBy})
		}
		if err != nil {
			dao.GetLogger().Error("Error in GetObjectTypeByName", zap.Error(err))
		}
	} else {
		tx.Commit()
		if objectType.IsDeleted && addIfMissing {
			objectType, err = dao.CreateObjectType(&models.ODObjectType{Name: typeName, CreatedBy: createdBy})
		}
		if err != nil {
			dao.GetLogger().Error("Error in GetObjectTypeByName", zap.Error(err))
		}
	}
	return objectType, err
}

func getObjectTypeByNameInTransaction(tx *sqlx.Tx, typeName string) (models.ODObjectType, error) {
	var objectType models.ODObjectType
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
        ,ownedBy
        ,name
        ,description
        ,contentConnector
    from
        object_type
    where
        name = ?
    order by isDeleted asc, createdDate desc limit 1    
	`
	err := tx.Get(&objectType, getObjectTypeStatement, typeName)
	// Return response
	return objectType, err
}

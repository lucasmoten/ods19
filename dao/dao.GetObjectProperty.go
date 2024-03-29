package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetObjectProperty return the requested property by ID.
// NOTE: Should we just pass an ID instead?
func (dao *DataAccessLayer) GetObjectProperty(objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error) {
	defer util.Time("GetObjectProperty")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.ODObjectPropertyEx{}, err
	}
	dbObjectProperty, err := getObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectProperty", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectProperty, err
}

func getObjectPropertyInTransaction(tx *sqlx.Tx, objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error) {
	var dbObjectProperty models.ODObjectPropertyEx
	query := `
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
        ,propertyValue
        ,classificationPM     
    from property 
    where id = ?`
	err := tx.Get(&dbObjectProperty, query, objectProperty.ID)
	if err != nil {
		print(err.Error())
	}
	return dbObjectProperty, err
}

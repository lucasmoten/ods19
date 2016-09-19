package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetPropertiesForObject retrieves the properties for a given object.
func (dao *DataAccessLayer) GetPropertiesForObject(object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return []models.ODObjectPropertyEx{}, err
	}
	response, err := getPropertiesForObjectInTransaction(tx, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetPropertiesForObject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getPropertiesForObjectInTransaction(tx *sqlx.Tx, object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
	query := `
    select
        p.id
        ,p.createdDate
        ,p.createdBy
        ,p.modifiedDate
        ,p.modifiedBy
        ,p.isDeleted
        ,p.deletedDate
        ,p.deletedBy
        ,p.changeCount
        ,p.changeToken
        ,p.name
        ,p.propertyValue
        ,p.classificationPM     
    from property p
        inner join object_property op on p.id = op.propertyid
    where 
        p.isdeleted = 0 
        and op.isdeleted = 0 
        and op.objectid = ?`
	err := tx.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

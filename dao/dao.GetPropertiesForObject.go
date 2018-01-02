package dao

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetPropertiesForObject retrieves the properties for a given object.
func (dao *DataAccessLayer) GetPropertiesForObject(object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	defer util.Time("GetPropertiesForObject")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return []models.ODObjectPropertyEx{}, err
	}
	response, err := getPropertiesForObjectInTransaction(tx, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetPropertiesForObject", zap.Error(err))
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
		and op.objectid = ?
	order by
		p.name asc, p.propertyValue asc
		`
	err := tx.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

package dao

import (
	"strconv"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetPropertiesForObjectRevision retrieves the properties for a specific
// revision of the given object instead of the current revision.
func (dao *DataAccessLayer) GetPropertiesForObjectRevision(object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	defer util.Time("GetPropertiesForObjectRevision")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return []models.ODObjectPropertyEx{}, err
	}
	response, err := getPropertiesForObjectRevisionInTransaction(tx, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetPropertiesForObjectRevision", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getPropertiesForObjectRevisionInTransaction(tx *sqlx.Tx, object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
	// #989 There is no archive table for the join betwen a_object and a_property, so need to use the object_property
	// table for the join, and constrain by the date for those properties created or modified no later than 10
	// milliseconds after the object modification time. Previously this was 1 full second, which is far too generous
	query := `
    select 
        ap.id
        ,ap.createdDate
        ,ap.createdBy
        ,ap.modifiedDate
        ,ap.modifiedBy
        ,ap.isDeleted
        ,ap.deletedDate
        ,ap.deletedBy
        ,ap.changeCount
        ,ap.changeToken
        ,ap.name
        ,ap.propertyValue
        ,ap.classificationPM  
    from a_property ap
        inner join object_property op on ap.id = op.propertyid
        inner join a_object ao on op.objectid = ao.id
    where 
        ap.isdeleted = 0 
        and ap.modifiedDate < date_add(ao.modifieddate,interval ` + strconv.Itoa(updateTimeWindowMS*1000) + ` microsecond)
        and (op.isdeleted = 0 or op.deleteddate > date_add(ao.modifieddate,interval ` + strconv.Itoa(updateTimeWindowMS*1000) + ` microsecond))
        and ao.id = ?
		and ao.changeCount = ?
	order by
		ap.name asc, ap.propertyValue asc
		`
	err := tx.Select(&response, query, object.ID, object.ChangeCount)
	if err != nil {
		return response, err
	}
	return response, err
}

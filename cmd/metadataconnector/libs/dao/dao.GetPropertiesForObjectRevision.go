package dao

import (
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetPropertiesForObjectRevision retrieves the properties for a specific
// revision of the given object instead of the current revision.
func (dao *DataAccessLayer) GetPropertiesForObjectRevision(object models.ODObject) ([]models.ODObjectPropertyEx, error) {

	tx := dao.MetadataDB.MustBegin()
	response, err := getPropertiesForObjectRevisionInTransaction(tx, object)
	if err != nil {
		log.Printf("Error in GetPropertiesForObjectRevision: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getPropertiesForObjectRevisionInTransaction(tx *sqlx.Tx, object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
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
        and ap.createdDate < date_add(ao.modifieddate,interval 1 minute)
        and op.isdeleted = 0 
        and ao.id = ?
        and ao.changeCount = ?`
	err := tx.Select(&response, query, object.ID, object.ChangeCount)
	if err != nil {
		return response, err
	}
	return response, err
}

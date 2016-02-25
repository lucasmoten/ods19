package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetPropertiesForObject retrieves the properties for a given object.
func (dao *DataAccessLayer) GetPropertiesForObject(object *models.ODObject) ([]models.ODObjectPropertyEx, error) {

	tx := dao.MetadataDB.MustBegin()
	response, err := getPropertiesForObjectInTransaction(tx, object)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func getPropertiesForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) ([]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
	query := `select p.* from property p
            inner join object_property op on p.id = op.propertyid
            where p.isdeleted = 0 and op.isdeleted = 0 and op.objectid = ?`
	err := tx.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

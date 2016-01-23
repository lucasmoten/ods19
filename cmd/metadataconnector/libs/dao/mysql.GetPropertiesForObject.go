package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetPropertiesForObject retrieves the properties for a given object
func GetPropertiesForObject(db *sqlx.DB, objectID []byte) (*[]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
	query := `select p.* from property p inner join object_property op on p.id = op.propertyid where p.isdeleted = 0 and op.isdeleted = 0 and op.objectid = ?`
	err := db.Select(&response, query, objectID)
	if err != nil {
		return &response, err
	}
	return &response, err
}

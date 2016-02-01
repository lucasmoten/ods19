package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetPermissionsForObject retrieves the grants for a given object
func GetPermissionsForObject(db *sqlx.DB, object *models.ODObject) ([]models.ODObjectPermission, error) {
	response := []models.ODObjectPermission{}
	query := `select op.* from pbject_permission op inner join object o on op.objectid = o.objectid where op.isdeleted = 0 and op.objectid = ?`
	err := db.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

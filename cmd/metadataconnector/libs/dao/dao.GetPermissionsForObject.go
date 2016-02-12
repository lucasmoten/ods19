package dao

import "decipher.com/oduploader/metadata/models"

// GetPermissionsForObject retrieves the grants for a given object.
func (dao *DataAccessLayer) GetPermissionsForObject(object *models.ODObject) ([]models.ODObjectPermission, error) {

	response := []models.ODObjectPermission{}
	query := `select op.* from object_permission op inner join object o on op.objectid = o.id where op.isdeleted = 0 and op.objectid = ?`
	err := dao.MetadataDB.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

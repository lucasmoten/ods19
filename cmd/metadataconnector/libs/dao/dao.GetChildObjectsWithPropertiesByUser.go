package dao

import "decipher.com/oduploader/metadata/models"

// GetChildObjectsWithPropertiesByUser retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified object by
// parentID and are owned by the specified user or group.
func (dao *DataAccessLayer) GetChildObjectsWithPropertiesByUser(
	orderByClause string, pageNumber int, pageSize int, object *models.ODObject, user string) (models.ODObjectResultset, error) {
	response, err := dao.GetChildObjectsByUser(orderByClause, pageNumber, pageSize, object, user)
	if err != nil {
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := dao.GetPropertiesForObject(&response.Objects[i])
		if err != nil {
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

package dao

import "decipher.com/oduploader/metadata/models"

// GetChildObjectsWithPropertiesByOwner retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the specified object by
// parentID and are owned by the specified user or group.
func (dao *DataAccessLayer) GetChildObjectsWithPropertiesByOwner(
	orderByClause string, pageNumber int, pageSize int, object *models.ODObject, owner string) (models.ODObjectResultset, error) {
	response, err := dao.GetChildObjectsByOwner(orderByClause, pageNumber, pageSize, object, owner)
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

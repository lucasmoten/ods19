package dao

import "decipher.com/oduploader/metadata/models"

// GetChildObjectsWithProperties retrieves a list of Objects and their
// Properties in Object Drive that are nested beneath the parent object.
func (dao *DataAccessLayer) GetChildObjectsWithProperties(
	orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error) {

	response, err := dao.GetChildObjects(orderByClause, pageNumber, pageSize, object)
	if err != nil {
		print(err.Error())
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

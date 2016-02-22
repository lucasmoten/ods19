package dao

import "decipher.com/oduploader/metadata/models"

// GetRootObjectsWithPropertiesByUser retrieves a list of Objects and their
// Properties in Object Drive that are not nested beneath any other objects
// natively (natural parentId is null) and are owned by the specified user or
// group.
func (dao *DataAccessLayer) GetRootObjectsWithPropertiesByUser(
	orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {

	response, err := dao.GetRootObjectsByUser(orderByClause, pageNumber, pageSize, user)
	if err != nil {
		print(err.Error())
		return response, err
	}
	for i := 0; i < len(response.Objects); i++ {
		properties, err := dao.GetPropertiesForObject(&response.Objects[i])
		if err != nil {
			print(err.Error())
			return response, err
		}
		response.Objects[i].Properties = properties
	}
	return response, err
}

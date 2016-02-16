package dao

import "decipher.com/oduploader/metadata/models"

// GetRootObjectsWithProperties retrieves a list of Objects and their Properties
// in Object Drive that are not nested beneath any other objects natively
// (natural parentId is null)
func (dao *DataAccessLayer) GetRootObjectsWithProperties(
	orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {

	response, err := dao.GetRootObjects(orderByClause, pageNumber, pageSize)
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

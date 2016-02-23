package dao

import "decipher.com/oduploader/metadata/models"

// GetObject uses the passed in object and makes the appropriate sql calls to
// the database to retrieve and return the requested object by ID. Optionally,
// loadProperties flag pulls in nested properties associated with the object.
func (dao *DataAccessLayer) GetObject(object *models.ODObject, loadProperties bool) (*models.ODObject, error) {
	var dbObject models.ODObject
	getObjectStatement := `select o.*, ot.name typeName from object o inner join object_type ot on o.typeid = ot.id where o.id = ?`
	err := dao.MetadataDB.Get(&dbObject, getObjectStatement, object.ID)
	if err != nil {
		return &dbObject, err
	}

	dbObject.Permissions, err = dao.GetPermissionsForObject(object)
	if err != nil {
		return &dbObject, err
	}

	// Load properties if requested
	if loadProperties {
		dbObject.Properties, err = dao.GetPropertiesForObject(object)
		if err != nil {
			return &dbObject, err
		}
	}

	// All ready ....
	return &dbObject, err
}

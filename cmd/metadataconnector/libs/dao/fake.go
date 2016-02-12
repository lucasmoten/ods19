package dao

import "decipher.com/oduploader/metadata/models"

// FakeDAO is suitable for tests. Add fields to this struct to hold fake
// reponses for each of the methods that FakeDAO will implement. These fake
// response fields can be explicitly set in tests.
type FakeDAO struct {
	Err               error
	Object            *models.ODObject
	ObjectPermissions []models.ODObjectPermission
	ObjectProperites  []models.ODObjectPropertyEx
	// TODO: all required responses should be fields.
}

// AddPerAddPermissionToObject for FakeDAO.
func (fake *FakeDAO) AddPerAddPermissionToObject(createdBy string, object *models.ODObject, permission *models.ODObjectPermission) error {
	return fake.Err
}

// GetObject for FakeDAO.
func (fake *FakeDAO) GetObject(object *models.ODObject, loadProperties bool) (*models.ODObject, error) {
	// return what we set on the field
	return fake.Object, nil
}

// GetPermissionsForObject for FakeDAO.
func (fake *FakeDAO) GetPermissionsForObject(object *models.ODObject) ([]models.ODObjectPermission, error) {
	return fake.ObjectPermissions, nil
}

// GetPropertiesForObject for FakeDAO.
func (fake *FakeDAO) GetPropertiesForObject(object *models.ODObject) ([]models.ODObjectPropertyEx, error) {
	return fake.ObjectProperites, nil
}

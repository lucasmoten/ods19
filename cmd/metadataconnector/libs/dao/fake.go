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
	ObjectType        models.ODObjectType
	// TODO: all required responses should be fields.
}

// AddPermissionToObject for FakeDAO.
func (fake *FakeDAO) AddPermissionToObject(createdBy string, object *models.ODObject, permission *models.ODObjectPermission) error {
	return fake.Err
}

// AddPropertyToObject for FakeDAO.
func (fake *FakeDAO) AddPropertyToObject(createdBy string, object *models.ODObject, property *models.ODProperty) error {
	return fake.Err
}

// CreateObject for FakeDAO.
func (fake *FakeDAO) CreateObject(object *models.ODObject, acm *models.ODACM) error {
	fake.Object = object
	return fake.Err
}

// CreateObjectType for FakeDAO.
func (fake *FakeDAO) CreateObjectType(objectType *models.ODObjectType) error {
	fake.ObjectType = *objectType
	return fake.Err
}

// GetObject for FakeDAO.
func (fake *FakeDAO) GetObject(object *models.ODObject, loadProperties bool) (*models.ODObject, error) {
	// return what we set on the field
	return fake.Object, nil
}

// GetPermissionsForObject for FakeDAO.
func (fake *FakeDAO) GetPermissionsForObject(object *models.ODObject) ([]models.ODObjectPermission, error) {
	return fake.ObjectPermissions, fake.Err
}

// GetObjectTypeByName for FakeDAO.
func (fake *FakeDAO) GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {
	return fake.ObjectType, fake.Err
}

// GetPropertiesForObject for FakeDAO.
func (fake *FakeDAO) GetPropertiesForObject(object *models.ODObject) ([]models.ODObjectPropertyEx, error) {
	return fake.ObjectProperites, nil
}

func (fake *FakeDAO) clearError() {
	fake.Err = nil
}

// TODO: remove this. This is just to make the compiler mad when I leave off methods.
func getFakeDAO() DAO {
	return &FakeDAO{}
}

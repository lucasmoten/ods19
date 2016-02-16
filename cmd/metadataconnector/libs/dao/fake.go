package dao

import "decipher.com/oduploader/metadata/models"

// FakeDAO is suitable for tests. Add fields to this struct to hold fake
// reponses for each of the methods that FakeDAO will implement. These fake
// response fields can be explicitly set, or setup functions can be defined.
type FakeDAO struct {
	Err               error
	IsDescendent      bool
	Object            *models.ODObject
	ObjectPermissions []models.ODObjectPermission
	ObjectProperites  []models.ODObjectPropertyEx
	ObjectProperty    *models.ODObjectPropertyEx
	ObjectType        models.ODObjectType
	ObjectResultSet   models.ODObjectResultset
	User              *models.ODUser
	Users             []string
	// TODO: More fields required?
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
	return fake.Err
}

// CreateObjectType for FakeDAO.
func (fake *FakeDAO) CreateObjectType(objectType *models.ODObjectType) error {
	return fake.Err
}

// CreateUser for FakeDAO.
func (fake *FakeDAO) CreateUser(user *models.ODUser) (*models.ODUser, error) {
	return fake.User, fake.Err
}

// DeleteObject for FakeDAO.
func (fake *FakeDAO) DeleteObject(object *models.ODObject, explicit bool) error {
	return fake.Err
}

// DeleteObjectProperty for FakeDAO.
func (fake *FakeDAO) DeleteObjectProperty(objectProperty *models.ODObjectPropertyEx) error {
	return fake.Err
}

// DeleteObjectType for FakeDAO.
func (fake *FakeDAO) DeleteObjectType(objectType *models.ODObjectType) error {
	return fake.Err
}

// GetChildObjects for FakeDAO.
func (fake *FakeDAO) GetChildObjects(orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsByOwner for FakeDAO.
func (fake *FakeDAO) GetChildObjectsByOwner(
	orderByClause string, pageNumber int, pageSize int, object *models.ODObject, owner string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsWithProperties for FakeDAO.
func (fake *FakeDAO) GetChildObjectsWithProperties(
	orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsWithPropertiesByOwner for FakeDAO.
func (fake *FakeDAO) GetChildObjectsWithPropertiesByOwner(
	orderByClause string, pageNumber int, pageSize int, object *models.ODObject, owner string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetUserByDistinguishedName for FakeDAO.
func (fake *FakeDAO) GetUserByDistinguishedName(user *models.ODUser) (*models.ODUser, error) {
	return fake.User, fake.Err

}

// GetUsers for FakeDAO.
func (fake *FakeDAO) GetUsers() ([]string, error) {
	return fake.Users, fake.Err
}

// GetObject for FakeDAO.
func (fake *FakeDAO) GetObject(object *models.ODObject, loadProperties bool) (*models.ODObject, error) {
	return fake.Object, fake.Err
}

// GetObjectProperty for FakeDAO.
func (fake *FakeDAO) GetObjectProperty(objectProperty *models.ODObjectPropertyEx) (*models.ODObjectPropertyEx, error) {
	return fake.ObjectProperty, fake.Err
}

// GetObjectType for FakeDAO.
func (fake *FakeDAO) GetObjectType(objectType *models.ODObjectType) (*models.ODObjectType, error) {
	return &fake.ObjectType, fake.Err
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

// GetRootObjects for FakeDAO.
func (fake *FakeDAO) GetRootObjects(orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsByOwner for FakeDAO.
func (fake *FakeDAO) GetRootObjectsByOwner(
	orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsWithProperties for FakeDAO.
func (fake *FakeDAO) GetRootObjectsWithProperties(
	orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsWithPropertiesByOwner for FakeDAO.
func (fake *FakeDAO) GetRootObjectsWithPropertiesByOwner(
	orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// IsParentIDADescendent for FakeDAO.
func (fake *FakeDAO) IsParentIDADescendent(id []byte, parentID []byte) (bool, error) {
	return fake.IsDescendent, fake.Err
}

// UpdateObject for FakeDAO.
func (fake *FakeDAO) UpdateObject(object *models.ODObject, acm *models.ODACM) error {
	return fake.Err
}

// UpdateObjectProperty for FakeDAO.
func (fake *FakeDAO) UpdateObjectProperty(objectProperty *models.ODObjectPropertyEx) error {
	return fake.Err
}

// UpdatePermission for FakeDAO.
func (fake *FakeDAO) UpdatePermission(permission *models.ODObjectPermission) error {
	return fake.Err
}

func (fake *FakeDAO) clearError() {
	fake.Err = nil
}

func fakeCompileCheck() DAO {
	return &FakeDAO{}
}

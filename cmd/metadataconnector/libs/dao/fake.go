package dao

import (
	"errors"

	"decipher.com/oduploader/metadata/models"
)

// FakeDAO is suitable for tests. Add fields to this struct to hold fake
// reponses for each of the methods that FakeDAO will implement. These fake
// response fields can be explicitly set, or setup functions can be defined.
type FakeDAO struct {
	Err               error
	IsDescendent      bool
	Object            models.ODObject
	ObjectPermission  models.ODObjectPermission
	ObjectPermissions []models.ODObjectPermission
	ObjectProperites  []models.ODObjectPropertyEx
	ObjectPropertyEx  models.ODObjectPropertyEx
	ObjectType        models.ODObjectType
	ObjectResultSet   models.ODObjectResultset
	Property          models.ODProperty
	User              models.ODUser
	Users             []models.ODUser
	// TODO: More fields required?
}

// AddPermissionToObject for FakeDAO.
func (fake *FakeDAO) AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission, propagateToChildren bool, masterKey string) (models.ODObjectPermission, error) {
	return fake.ObjectPermission, fake.Err
}

// AddPropertyToObject for FakeDAO.
func (fake *FakeDAO) AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error) {
	return fake.Property, fake.Err
}

// CreateObject for FakeDAO.
func (fake *FakeDAO) CreateObject(object *models.ODObject, acm *models.ODACM) (models.ODObject, error) {
	return fake.Object, fake.Err
}

// CreateObjectType for FakeDAO.
func (fake *FakeDAO) CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error) {
	return fake.ObjectType, fake.Err
}

// CreateUser for FakeDAO.
func (fake *FakeDAO) CreateUser(user models.ODUser) (models.ODUser, error) {
	return fake.User, fake.Err
}

// DeleteObject for FakeDAO.
func (fake *FakeDAO) DeleteObject(object models.ODObject, explicit bool) error {
	return fake.Err
}

// DeleteObjectPermission for FakeDAO.
func (fake *FakeDAO) DeleteObjectPermission(objectPermission models.ODObjectPermission, propagateToChildren bool) (models.ODObjectPermission, error) {
	return fake.ObjectPermission, fake.Err
}

// DeleteObjectProperty for FakeDAO.
func (fake *FakeDAO) DeleteObjectProperty(objectProperty models.ODObjectPropertyEx) error {
	return fake.Err
}

// DeleteObjectType for FakeDAO.
func (fake *FakeDAO) DeleteObjectType(objectType models.ODObjectType) error {
	return fake.Err
}

// ExpungeObject for FakeDAO.
func (fake *FakeDAO) ExpungeObject(object models.ODObject, explicit bool) error {
	return fake.Err
}

// GetChildObjects for FakeDAO.
func (fake *FakeDAO) GetChildObjects(orderByClause string, pageNumber int, pageSize int, object models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsByUser for FakeDAO.
func (fake *FakeDAO) GetChildObjectsByUser(
	orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsWithProperties for FakeDAO.
func (fake *FakeDAO) GetChildObjectsWithProperties(
	orderByClause string, pageNumber int, pageSize int, object models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsWithPropertiesByUser for FakeDAO.
func (fake *FakeDAO) GetChildObjectsWithPropertiesByUser(
	orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetUserByDistinguishedName for FakeDAO.
func (fake *FakeDAO) GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error) {
	for _, u := range fake.Users {
		if user.DistinguishedName == u.DistinguishedName {
			u.ModifiedBy = u.DistinguishedName
			return u, nil
		}
	}
	return fake.User, errors.New("DistinguishedName not found in fake.Users slice. Did you set it on the fake?")

}

// GetUsers for FakeDAO.
func (fake *FakeDAO) GetUsers() ([]models.ODUser, error) {
	return fake.Users, fake.Err
}

// GetObject for FakeDAO.
func (fake *FakeDAO) GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error) {
	return fake.Object, fake.Err
}

// GetObjectPermission for FakeDAO.
func (fake *FakeDAO) GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
	return fake.ObjectPermission, fake.Err
}

// GetObjectProperty for FakeDAO.
func (fake *FakeDAO) GetObjectProperty(objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error) {
	return fake.ObjectPropertyEx, fake.Err
}

// GetObjectRevisionsByUser for FakeDAO
func (fake *FakeDAO) GetObjectRevisionsByUser(orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetObjectRevisionsWithPropertiesByUser for FakeDAO
func (fake *FakeDAO) GetObjectRevisionsWithPropertiesByUser(orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetObjectType for FakeDAO.
func (fake *FakeDAO) GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error) {
	return &fake.ObjectType, fake.Err
}

// GetObjectsIHaveShared for FakeDAO
func (fake *FakeDAO) GetObjectsIHaveShared(orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetPermissionsForObject for FakeDAO.
func (fake *FakeDAO) GetPermissionsForObject(object models.ODObject) ([]models.ODObjectPermission, error) {
	return fake.ObjectPermissions, fake.Err
}

// GetObjectTypeByName for FakeDAO.
func (fake *FakeDAO) GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {
	return fake.ObjectType, fake.Err
}

// GetPropertiesForObject for FakeDAO.
func (fake *FakeDAO) GetPropertiesForObject(object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	return fake.ObjectProperites, nil
}

// GetPropertiesForObjectRevision for FakeDAO
func (fake *FakeDAO) GetPropertiesForObjectRevision(object models.ODObject) ([]models.ODObjectPropertyEx, error) {
	return fake.ObjectProperites, nil
}

// GetRootObjects for FakeDAO.
func (fake *FakeDAO) GetRootObjects(orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsByUser for FakeDAO.
func (fake *FakeDAO) GetRootObjectsByUser(
	orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsWithProperties for FakeDAO.
func (fake *FakeDAO) GetRootObjectsWithProperties(
	orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsWithPropertiesByUser for FakeDAO.
func (fake *FakeDAO) GetRootObjectsWithPropertiesByUser(
	orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetTrashedObjectsByUser for FakeDAO.
func (fake *FakeDAO) GetTrashedObjectsByUser(orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error) {
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
func (fake *FakeDAO) UpdateObjectProperty(objectProperty models.ODObjectPropertyEx) error {
	return fake.Err
}

// UpdatePermission for FakeDAO.
func (fake *FakeDAO) UpdatePermission(permission models.ODObjectPermission) error {
	return fake.Err
}

func (fake *FakeDAO) clearError() {
	fake.Err = nil
}

func fakeCompileCheck() DAO {
	return &FakeDAO{}
}

// GetObjectsSharedToMe gives a listing of files that were shared to us
func (fake *FakeDAO) GetObjectsSharedToMe(
	grantee string,
	orderByClause string,
	pageNumber int,
	pageSize int,
) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

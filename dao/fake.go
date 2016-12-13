package dao

import (
	"errors"

	"github.com/uber-go/zap"

	"time"

	globalconfig "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
)

// FakeDAO is suitable for tests. Add fields to this struct to hold fake
// reponses for each of the methods that FakeDAO will implement. These fake
// response fields can be explicitly set, or setup functions can be defined.
type FakeDAO struct {
	AcmGrantee        models.ODAcmGrantee
	AcmGrantees       []models.ODAcmGrantee
	DBState           models.DBState
	Err               error
	IsDescendent      bool
	Object            models.ODObject
	ObjectPermission  models.ODObjectPermission
	ObjectPermissions []models.ODObjectPermission
	ObjectProperites  []models.ODObjectPropertyEx
	ObjectPropertyEx  models.ODObjectPropertyEx
	ObjectType        models.ODObjectType
	ObjectResultSet   models.ODObjectResultset
	Parents           []models.ODObject
	Property          models.ODProperty
	User              models.ODUser
	Users             []models.ODUser
	UserStatsData     models.UserStats
}

// AddPermissionToObject for FakeDAO.
func (fake *FakeDAO) AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error) {
	return fake.ObjectPermission, fake.Err
}

// AddPropertyToObject for FakeDAO.
func (fake *FakeDAO) AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error) {
	return fake.Property, fake.Err
}

// CreateObject for FakeDAO.
func (fake *FakeDAO) CreateObject(object *models.ODObject) (models.ODObject, error) {
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
func (fake *FakeDAO) DeleteObject(user models.ODUser, object models.ODObject, explicit bool) error {
	return fake.Err
}

// DeleteObjectPermission for FakeDAO.
func (fake *FakeDAO) DeleteObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
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

// Expunge objects deleted by user.
func (fake *FakeDAO) ExpungeDeletedByUser(user models.ODUser, pageSize int) (int64, error) {
	return int64(0), fake.Err
}

// ExpungeObject for FakeDAO.
func (fake *FakeDAO) ExpungeObject(user models.ODUser, object models.ODObject, explicit bool) error {
	return fake.Err
}

// GetAcmGrantee for FakeDAO
func (fake *FakeDAO) GetAcmGrantee(grantee string) (models.ODAcmGrantee, error) {
	return fake.AcmGrantee, fake.Err
}

// GetAcmGrantees for FakeDAO
func (fake *FakeDAO) GetAcmGrantees(grantees []string) ([]models.ODAcmGrantee, error) {
	return fake.AcmGrantees, fake.Err
}

// GetChildObjects for FakeDAO.
func (fake *FakeDAO) GetChildObjects(pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsByUser for FakeDAO.
func (fake *FakeDAO) GetChildObjectsByUser(
	user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsWithProperties for FakeDAO.
func (fake *FakeDAO) GetChildObjectsWithProperties(
	pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetChildObjectsWithPropertiesByUser for FakeDAO.
func (fake *FakeDAO) GetChildObjectsWithPropertiesByUser(
	user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetDBState for FakeDAO
func (fake *FakeDAO) GetDBState() (models.DBState, error) {
	fake.DBState.SchemaVersion = SchemaVersion
	fake.DBState.Identifier = "fake"
	fake.DBState.CreateDate = time.Now()
	return fake.DBState, fake.Err
}

// GetLogger returns a logger for the current session (or any other context - we want correlation across a request)
func (fake *FakeDAO) GetLogger() zap.Logger {
	return globalconfig.RootLogger
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

// GetObjectRevision for FakeDAO.
func (fake *FakeDAO) GetObjectRevision(object models.ODObject, loadProperties bool) (models.ODObject, error) {
	return fake.Object, fake.Err
}

// GetObjectRevisionsByUser for FakeDAO
func (fake *FakeDAO) GetObjectRevisionsByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetObjectRevisionsWithPropertiesByUser for FakeDAO
func (fake *FakeDAO) GetObjectRevisionsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetObjectType for FakeDAO.
func (fake *FakeDAO) GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error) {
	return &fake.ObjectType, fake.Err
}

// GetObjectTypeByName for FakeDAO.
func (fake *FakeDAO) GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {
	return fake.ObjectType, fake.Err
}

// GetObjectsIHaveShared for FakeDAO
func (fake *FakeDAO) GetObjectsIHaveShared(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetObjectsSharedToEveryone for FakeDAO
func (fake *FakeDAO) GetObjectsSharedToEveryone(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetObjectsSharedToMe for FakeDAO
func (fake *FakeDAO) GetObjectsSharedToMe(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetParents for FakeDAO
func (fake *FakeDAO) GetParents(child models.ODObject) ([]models.ODObject, error) {
	return fake.Parents, fake.Err
}

// GetPermissionsForObject for FakeDAO.
func (fake *FakeDAO) GetPermissionsForObject(object models.ODObject) ([]models.ODObjectPermission, error) {
	return fake.ObjectPermissions, fake.Err
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
func (fake *FakeDAO) GetRootObjects(pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsByGroup for FakeDAO
func (fake *FakeDAO) GetRootObjectsByGroup(groupName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsByUser for FakeDAO.
func (fake *FakeDAO) GetRootObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsWithProperties for FakeDAO.
func (fake *FakeDAO) GetRootObjectsWithProperties(pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsWithPropertiesByGroup for FakeDAO.
func (fake *FakeDAO) GetRootObjectsWithPropertiesByGroup(groupName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetRootObjectsWithPropertiesByUser for FakeDAO.
func (fake *FakeDAO) GetRootObjectsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// GetTrashedObjectsByUser for FakeDAO.
func (fake *FakeDAO) GetTrashedObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
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

// GetUserStats is per user statistics
func (fake *FakeDAO) GetUserStats(dn string) (models.UserStats, error) {
	return fake.UserStatsData, fake.Err
}

// IsParentIDADescendent for FakeDAO.
func (fake *FakeDAO) IsParentIDADescendent(id []byte, parentID []byte) (bool, error) {
	return fake.IsDescendent, fake.Err
}

// SearchObjectsByNameOrDescription for FakeDAO
func (fake *FakeDAO) SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest PagingRequest, loadProperties bool) (models.ODObjectResultset, error) {
	return fake.ObjectResultSet, fake.Err
}

// UndeleteObject for FakeDAO.
func (fake *FakeDAO) UndeleteObject(object *models.ODObject) (models.ODObject, error) {
	return fake.Object, fake.Err
}

// UpdateObject for FakeDAO.
func (fake *FakeDAO) UpdateObject(object *models.ODObject) error {
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

// fakeCompileCheck ensures that FakeDAO implements DAO.
func fakeCompileCheck() DAO {
	return &FakeDAO{}
}

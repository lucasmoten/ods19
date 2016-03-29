package dao

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"github.com/jmoiron/sqlx"
)

// SchemaVersion marks compatibility with previously created databases.
// On startup, we should be checking the schema, and raise some alarm if
// the schema is out of date, or trigger a migration, etc.
//
// This is also here so that the database instance is uniquely identified so that
// the S3 buckets partition in a way that allows us to know which S3 files
// go with what instance.
var SchemaVersion = "20160314"

// DAO defines the contract our app has with the database.
type DAO interface {
	AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission, propagateToChildren bool, masterKey string) (models.ODObjectPermission, error)
	AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error)
	CreateObject(object *models.ODObject) (models.ODObject, error)
	CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error)
	CreateUser(models.ODUser) (models.ODUser, error)
	DeleteObject(object models.ODObject, explicit bool) error
	DeleteObjectPermission(objectPermission models.ODObjectPermission, propagateToChildren bool) (models.ODObjectPermission, error)
	DeleteObjectProperty(objectProperty models.ODObjectPropertyEx) error
	DeleteObjectType(objectType models.ODObjectType) error
	ExpungeObject(object models.ODObject, explicit bool) error
	GetChildObjects(pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithProperties(pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithPropertiesByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetDBState() (models.DBState, error)
	GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error)
	GetObjectProperty(objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error)
	GetObjectRevision(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectRevisionsByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetObjectRevisionsWithPropertiesByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error)
	GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error)
	GetObjectsIHaveShared(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetObjectsSharedToMe(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetPermissionsForObject(object models.ODObject) ([]models.ODObjectPermission, error)
	GetPropertiesForObject(object models.ODObject) ([]models.ODObjectPropertyEx, error)
	GetPropertiesForObjectRevision(object models.ODObject) ([]models.ODObjectPropertyEx, error)
	GetRootObjects(pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsByUser(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithProperties(pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithPropertiesByUser(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetTrashedObjectsByUser(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error)
	GetUsers() ([]models.ODUser, error)
	IsParentIDADescendent(id []byte, parentID []byte) (bool, error)
	SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest protocol.PagingRequest, loadProperties bool) (models.ODObjectResultset, error)
	UndeleteObject(object *models.ODObject) (models.ODObject, error)
	UpdateObject(object *models.ODObject) error
	UpdateObjectProperty(objectProperty models.ODObjectPropertyEx) error
	UpdatePermission(permission models.ODObjectPermission) error
}

// DataAccessLayer is a concrete DAO implementation with a true DB connection.
type DataAccessLayer struct {
	MetadataDB *sqlx.DB
}

func daoCompileCheck() DAO {
	// function exists to make compiler complain when interface changes.
	return &DataAccessLayer{}
}

package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// DAO defines the contract our app has with the database.
type DAO interface {
	AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission, propagateToChildren bool, masterKey string) (models.ODObjectPermission, error)
	AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error)
	CreateObject(object *models.ODObject, acm *models.ODACM) (models.ODObject, error)
	CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error)
	CreateUser(models.ODUser) (models.ODUser, error)
	DeleteObject(object models.ODObject, explicit bool) error
	DeleteObjectPermission(objectPermission models.ODObjectPermission, propagateToChildren bool) (models.ODObjectPermission, error)
	DeleteObjectProperty(objectProperty models.ODObjectPropertyEx) error
	DeleteObjectType(objectType models.ODObjectType) error
	ExpungeObject(object models.ODObject, explicit bool) error
	GetChildObjects(orderByClause string, pageNumber int, pageSize int, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsByUser(orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error)
	GetChildObjectsWithProperties(orderByClause string, pageNumber int, pageSize int, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithPropertiesByUser(orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error)
	GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error)
	GetObjectProperty(objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error)
	GetObjectRevisionsByUser(orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error)
	GetObjectRevisionsWithPropertiesByUser(orderByClause string, pageNumber int, pageSize int, object models.ODObject, user string) (models.ODObjectResultset, error)
	GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error)
	GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error)
	GetObjectsIHaveShared(orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error)
	GetObjectsSharedToMe(owner string, orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error)
	GetPermissionsForObject(object models.ODObject) ([]models.ODObjectPermission, error)
	GetPropertiesForObject(object models.ODObject) ([]models.ODObjectPropertyEx, error)
	GetPropertiesForObjectRevision(object models.ODObject) ([]models.ODObjectPropertyEx, error)
	GetRootObjects(orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error)
	GetRootObjectsByUser(orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error)
	GetRootObjectsWithProperties(orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error)
	GetRootObjectsWithPropertiesByUser(orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error)
	GetTrashedObjectsByUser(orderByClause string, pageNumber int, pageSize int, user string) (models.ODObjectResultset, error)
	GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error)
	GetUsers() ([]models.ODUser, error)
	IsParentIDADescendent(id []byte, parentID []byte) (bool, error)
	UpdateObject(object *models.ODObject, acm *models.ODACM) error
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

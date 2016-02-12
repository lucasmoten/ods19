package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// DAO defines the contract our app has with the database.
type DAO interface {
	AddPermissionToObject(createdBy string, object *models.ODObject, permission *models.ODObjectPermission) error
	AddPropertyToObject(createdBy string, object *models.ODObject, property *models.ODProperty) error
	CreateObject(object *models.ODObject, acm *models.ODACM) error
	CreateObjectType(objectType *models.ODObjectType) error
	// DeleteObject(object *models.ODObject, explicit bool) error
	// DeleteObjectProperty(objectProperty *models.ODObjectPropertyEx) error
	// DeleteObjectTypDeleteObjectType(objectType *models.ODObjectType) error
	// GetChildObjects(orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error)
	// GetChildObjectsByOwner(orderByClause string, pageNumber int, pageSize int, object *models.ODObject, owner string) (models.ODObjectResultset, error)
	// GetChildObjectsWithProperties(orderByClause string, pageNumber int, pageSize int, object *models.ODObject) (models.ODObjectResultset, error)
	// GetChildObjectsWithPropertiesByOwner(orderByClause string, pageNumber int, pageSize int, object *models.ODObject, owner string) (models.ODObjectResultset, error)
	GetObject(object *models.ODObject, loadProperties bool) (*models.ODObject, error)
	// GetObjectProperty(objectProperty *models.ODObjectPropertyEx) (*models.ODObjectPropertyEx, error)
	// GetObjectType(objectType *models.ODObjectType) (*models.ODObjectType, error)
	GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error)
	GetPermissionsForObject(object *models.ODObject) ([]models.ODObjectPermission, error)
	GetPropertiesForObject(object *models.ODObject) ([]models.ODObjectPropertyEx, error)
	// GetRootObjects(orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error)
	// GetRootObjectsByOwner(orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error)
	// GetRootObjectsWithProperties(orderByClause string, pageNumber int, pageSize int) (models.ODObjectResultset, error)
	// GetRootObjectsWithPropertiesByOwner(orderByClause string, pageNumber int, pageSize int, owner string) (models.ODObjectResultset, error)
	// GetUserByDistinguishedName(user *models.ODUser) (*models.ODUser, error)
	// GetUsers() ([]string, error)
	// UpdateObject(object *models.ODObject, acm *models.ODACM) error
	// UpdateObjectProperty(objectProperty *models.ODObjectPropertyEx) error
	// UpdatePermission(permission *models.ODObjectPermission) error
}

// DataAccessLayer is a concrete DAO implementation with a true db conn.
// Production servers should use this.
type DataAccessLayer struct {
	MetadataDB *sqlx.DB
}

// TODO: remove this. This is just to make the compiler mad when I leave off methods.
func getRealDAO() DAO {
	return &DataAccessLayer{}
}

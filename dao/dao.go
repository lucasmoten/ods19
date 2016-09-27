package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// SchemaVersion marks compatibility with previously created databases.
// On startup, we should be checking the schema, and raise some alarm if
// the schema is out of date, or trigger a migration, etc.
var SchemaVersion = "20160824"

// DAO defines the contract our app has with the database.
type DAO interface {
	AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission, propagateToChildren bool, masterKey string) (models.ODObjectPermission, error)
	AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error)
	CreateObject(object *models.ODObject) (models.ODObject, error)
	CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error)
	CreateUser(models.ODUser) (models.ODUser, error)
	DeleteObject(user models.ODUser, object models.ODObject, explicit bool) error
	DeleteObjectPermission(objectPermission models.ODObjectPermission, propagateToChildren bool) (models.ODObjectPermission, error)
	DeleteObjectProperty(objectProperty models.ODObjectPropertyEx) error
	DeleteObjectType(objectType models.ODObjectType) error
	ExpungeObject(user models.ODUser, object models.ODObject, explicit bool) error
	GetChildObjects(pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithProperties(pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithPropertiesByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetDBState() (models.DBState, error)
	GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error)
	GetObjectProperty(objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error)
	GetObjectRevision(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectRevisionsByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error)
	GetObjectRevisionsWithPropertiesByUser(user models.ODUser, pagingRequest protocol.PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error)
	GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error)
	GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error)
	GetObjectsIHaveShared(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetObjectsSharedToEveryone(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetObjectsSharedToMe(user models.ODUser, pagingRequest protocol.PagingRequest) (models.ODObjectResultset, error)
	GetParents(child models.ODObject) ([]models.ODObject, error)
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
	GetUserStats(dn string) (models.UserStats, error)
	IsParentIDADescendent(id []byte, parentID []byte) (bool, error)
	SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest protocol.PagingRequest, loadProperties bool) (models.ODObjectResultset, error)
	UndeleteObject(object *models.ODObject) (models.ODObject, error)
	UpdateObject(object *models.ODObject) error
	UpdateObjectProperty(objectProperty models.ODObjectPropertyEx) error
	UpdatePermission(permission models.ODObjectPermission) error
	GetLogger() zap.Logger
}

type CheckACM func(*models.ODObject) bool

// DataAccessLayer is a concrete DAO implementation with a true DB connection.
type DataAccessLayer struct {
	//This can be shared among structs
	MetadataDB *sqlx.DB
	//This can have a different value per http session
	Logger zap.Logger
}

// NewDerivedDAO is a dao bound to a new logger
func NewDerivedDAO(d DAO, logger zap.Logger) DAO {
	switch d2 := d.(type) {
	case *DataAccessLayer:
		return &DataAccessLayer{
			MetadataDB: d2.MetadataDB,
			Logger:     logger,
		}
	case *FakeDAO:
		return d
	}
	return nil
}

// GetLogger is a logger, probably for this session
func (dao *DataAccessLayer) GetLogger() zap.Logger {
	return dao.Logger
}

func daoCompileCheck() DAO {
	// function exists to make compiler complain when interface changes.
	return &DataAccessLayer{}
}

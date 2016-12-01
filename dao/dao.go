package dao

import (
	"fmt"
	"os"
	"time"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// SchemaVersion marks compatibility with previously created databases.
// On startup, we should be checking the schema, and raise some alarm if
// the schema is out of date, or trigger a migration, etc.
var SchemaVersion = "20160824"

// DAO defines the contract our app has with the database.
type DAO interface {
	AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error)
	AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error)
	CreateObject(object *models.ODObject) (models.ODObject, error)
	CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error)
	CreateUser(models.ODUser) (models.ODUser, error)
	DeleteObject(user models.ODUser, object models.ODObject, explicit bool) error
	DeleteObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error)
	DeleteObjectProperty(objectProperty models.ODObjectPropertyEx) error
	DeleteObjectType(objectType models.ODObjectType) error
	ExpungeObject(user models.ODUser, object models.ODObject, explicit bool) error
	GetAcmGrantee(grantee string) (models.ODAcmGrantee, error)
	GetAcmGrantees(grantees []string) ([]models.ODAcmGrantee, error)
	GetChildObjects(pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithProperties(pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetDBState() (models.DBState, error)
	GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error)
	GetObjectProperty(objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error)
	GetObjectRevision(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectRevisionsByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error)
	GetObjectRevisionsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject, checkACM CheckACM) (models.ODObjectResultset, error)
	GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error)
	GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error)
	GetObjectsIHaveShared(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetObjectsSharedToEveryone(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetObjectsSharedToMe(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetParents(child models.ODObject) ([]models.ODObject, error)
	GetPermissionsForObject(object models.ODObject) ([]models.ODObjectPermission, error)
	GetPropertiesForObject(object models.ODObject) ([]models.ODObjectPropertyEx, error)
	GetPropertiesForObjectRevision(object models.ODObject) ([]models.ODObjectPropertyEx, error)
	GetRootObjects(pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsByGroup(groupName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithProperties(pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithPropertiesByGroup(groupName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetTrashedObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error)
	GetUsers() ([]models.ODUser, error)
	GetUserStats(dn string) (models.UserStats, error)
	IsParentIDADescendent(id []byte, parentID []byte) (bool, error)
	SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest PagingRequest, loadProperties bool) (models.ODObjectResultset, error)
	UndeleteObject(object *models.ODObject) (models.ODObject, error)
	UpdateObject(object *models.ODObject) error
	UpdateObjectProperty(objectProperty models.ODObjectPropertyEx) error
	UpdatePermission(permission models.ODObjectPermission) error
	GetLogger() zap.Logger
}

// CheckACM is a function passed in for checking authorization to read an object
type CheckACM func(*models.ODObject) bool

// DataAccessLayer is a concrete DAO implementation with a true DB connection.
type DataAccessLayer struct {
	// MetadataDB is the connection.
	MetadataDB *sqlx.DB
	// Logger has a default, but can be updated by passing options to constructor.
	Logger zap.Logger
}

// Opt sets an option on DataAccessLayer.
type Opt func(*DataAccessLayer)

// WithLogger sets a custom logger on DataAccessLayer.
func WithLogger(logger zap.Logger) Opt {
	return func(d *DataAccessLayer) {
		d.Logger = logger
	}
}

// NewDataAccessLayer constructs a new DataAccessLayer with defaults and options. A string database
// identifier is also returned.
func NewDataAccessLayer(conf config.DatabaseConfiguration, opts ...Opt) (*DataAccessLayer, string, error) {

	db, err := conf.GetDatabaseHandle()
	if err != nil {
		return nil, "", err
	}
	d := DataAccessLayer{MetadataDB: db}

	defaults(&d)
	for _, opt := range opts {
		opt(&d)
	}

	err = pingDB(&d)
	if err != nil {
		return nil, "", fmt.Errorf("could not ping database: %v", err)
	}

	state, err := d.GetDBState()
	if err != nil {
		return nil, "", fmt.Errorf("getting db state failed: %v", err)
	}

	return &d, state.Identifier, nil
}

func defaults(d *DataAccessLayer) {
	d.Logger = zap.New(zap.NewJSONEncoder(), zap.Output(os.Stdout), zap.ErrorOutput(os.Stdout)).With(zap.String("node", config.NodeID))
}

// GetLogger is a logger, probably for this session
func (d *DataAccessLayer) GetLogger() zap.Logger {
	return d.Logger
}

func daoCompileCheck() DAO {
	// function exists to make compiler complain when interface changes.
	return &DataAccessLayer{}
}

func pingDB(d *DataAccessLayer) error {

	logger := d.GetLogger()

	attempts := 0
	max := 20
	sleep := 3

	var err error

	for attempts < max {

		attempts++

		err = d.MetadataDB.Ping()
		if err != nil {
			logger.Info("db sleep for retry")
			time.Sleep(time.Duration(sleep) * time.Second)
		} else {
			_, err = d.GetDBState()
			if err != nil {
				logger.Info("db available but schema not populated")
				time.Sleep(time.Duration(sleep) * time.Second)
			}
		}

	}
	return err
}

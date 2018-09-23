package dao

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// SchemaVersion marks compatibility with previously created databases.
// On startup, we should be checking the schema, and raise some alarm if
// the schema is out of date, or trigger a migration, etc.
var SchemaVersion = "20170726"
var mutexReadOnly sync.Mutex

// DAO defines the contract our app has with the database.
type DAO interface {
	AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error)
	AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error)
	AssociateUsersToNewACM(object models.ODObject, done chan bool) error
	CreateAcmGrantee(acmGrantee models.ODAcmGrantee) (models.ODAcmGrantee, error)
	CreateObject(object *models.ODObject) (models.ODObject, error)
	CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error)
	CreateUser(models.ODUser) (models.ODUser, error)
	DeleteObject(user models.ODUser, object models.ODObject, explicit bool) error
	DeleteObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error)
	DeleteObjectProperty(objectProperty models.ODObjectPropertyEx) error
	DeleteObjectType(objectType models.ODObjectType) error
	ExpungeDeletedByUser(user models.ODUser, pageSize int) (models.ODObjectResultset, error)
	ExpungeObject(user models.ODUser, object models.ODObject, explicit bool) error
	GetAcmGrantee(grantee string) (models.ODAcmGrantee, error)
	GetAcmGrantees(grantees []string) ([]models.ODAcmGrantee, error)
	GetChildObjects(pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithProperties(pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetChildObjectsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject) (models.ODObjectResultset, error)
	GetDBState() (models.DBState, error)
	GetGroupsForUser(user models.ODUser) (models.GroupSpaceResultset, error)
	GetLogger() *zap.Logger
	GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error)
	GetObjectProperty(objectProperty models.ODObjectPropertyEx) (models.ODObjectPropertyEx, error)
	GetObjectRevision(object models.ODObject, loadProperties bool) (models.ODObject, error)
	GetObjectRevisionsByUser(user models.ODUser, pagingRequest PagingRequest, object models.ODObject, withProperties bool) (models.ODObjectResultset, error)
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
	GetRootObjectsByGroup(groupGranteeName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithProperties(pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithPropertiesByGroup(groupGranteeName string, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetRootObjectsWithPropertiesByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetTrashedObjectsByUser(user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error)
	GetUserAOCacheByDistinguishedName(user models.ODUser) (models.ODUserAOCache, error)
	GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error)
	GetUsers() ([]models.ODUser, error)
	GetUserStats(dn string) (models.UserStats, error)
	IsParentIDADescendent(id []byte, parentID []byte) (bool, error)
	IsReadOnly(refresh bool) bool
	RebuildUserACMCache(useraocache *models.ODUserAOCache, user models.ODUser, done chan bool, mode string) error
	SearchObjectsByNameOrDescription(user models.ODUser, pagingRequest PagingRequest, loadProperties bool) (models.ODObjectResultset, error)
	SetUserAOCacheByDistinguishedName(useraocache *models.ODUserAOCache, user models.ODUser) error
	UndeleteObject(object *models.ODObject) (models.ODObject, error)
	UpdateObject(object *models.ODObject) error
	UpdateObjectProperty(objectProperty models.ODObjectPropertyEx) error
	UpdatePermission(permission models.ODObjectPermission) error
}

// DataAccessLayer is a concrete DAO implementation with a true DB connection.
type DataAccessLayer struct {
	// MetadataDB is the connection.
	MetadataDB *sqlx.DB
	// Logger has a default, but can be updated by passing options to constructor.
	Logger *zap.Logger
	// ReadOnly denotes whether database is accepting changes
	ReadOnly bool
	// SchemaVersion indicates the database schema version
	SchemaVersion string
	// Parameters to resolve deadlock needed by interface methods
	DeadlockRetryCounter int64
	DeadlockRetryDelay   int64
}

// Verify that DataAccessLayer Implements DAO.
var _ DAO = (*DataAccessLayer)(nil)

// Opt sets an option on DataAccessLayer.
type Opt func(*DataAccessLayer)

// WithLogger sets a custom logger on DataAccessLayer.
func WithLogger(logger *zap.Logger) Opt {
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

	defaults(&d, conf)
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
	d.SchemaVersion = state.SchemaVersion
	if d.SchemaVersion == SchemaVersion {
		d.ReadOnly = false
	}

	return &d, state.Identifier, nil
}

func defaults(d *DataAccessLayer, conf config.DatabaseConfiguration) {
	d.Logger = config.RootLogger
	d.ReadOnly = true
	d.DeadlockRetryCounter = conf.DeadlockRetryCounter
	d.DeadlockRetryDelay = conf.DeadlockRetryDelay
}

// GetLogger is a logger, probably for this session
func (d *DataAccessLayer) GetLogger() *zap.Logger {
	return d.Logger
}

func daoCompileCheck() DAO {
	// function exists to make compiler complain when interface changes.
	return &DataAccessLayer{}
}

func pingDB(d *DataAccessLayer) error {

	logger := d.GetLogger()

	attempts := 0
	max := 60
	sleep := 1

	var err error
	var state models.DBState

	for attempts < max {
		attempts++
		err = d.MetadataDB.Ping()
		if err != nil {
			logger.Info(fmt.Sprintf("database not yet available, rechecking in %d seconds", sleep))
		} else {
			state, err = d.GetDBState()
			switch {
			case err != nil:
				logger.Info(fmt.Sprintf("database online but schema not yet populated. rechecking in %d seconds", sleep))
			case state.SchemaVersion != SchemaVersion:
				logger.Info(fmt.Sprintf("database online with schema at version %s but expecting %s. rechecking in %d seconds for pending migration", state.SchemaVersion, SchemaVersion, sleep))
			case state.SchemaVersion == SchemaVersion:
				logger.Info(fmt.Sprintf("database online at schema version %s", SchemaVersion))
				sleep = 0
			}
		}
		if sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Second)
		} else {
			break
		}
	}
	return err
}

// IsReadOnly returns the current state of whether this DAO is considered read only
func (d *DataAccessLayer) IsReadOnly(refresh bool) bool {
	result := true
	mutexReadOnly.Lock()
	defer mutexReadOnly.Unlock()
	if refresh {
		beforeReadOnly := d.ReadOnly
		// Default to readonly
		d.ReadOnly = true
		// Find out our schema
		d.Logger.Debug("checking db state")
		state, err := d.GetDBState()
		if err != nil {
			d.Logger.Warn("getting db state failed", zap.Error(err))
		} else {
			d.SchemaVersion = state.SchemaVersion
			if d.SchemaVersion == SchemaVersion {
				d.ReadOnly = false
			}
			afterReadOnly := d.ReadOnly
			if beforeReadOnly != afterReadOnly {
				if afterReadOnly {
					d.Logger.Info(fmt.Sprintf("database online with schema at version %s but expecting %s. readonly = %t", d.SchemaVersion, SchemaVersion, afterReadOnly))
				} else {
					d.Logger.Info(fmt.Sprintf("database online with schema at version %s", d.SchemaVersion))
				}
			}
		}
	}
	result = d.ReadOnly
	return result
}

func containsAny(msg string, a []string) bool {
	for _, s := range a {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
func firstMatch(msg string, a []string) string {
	for _, s := range a {
		if strings.Contains(msg, s) {
			return s
		}
	}
	return ""

}

func execStatementWithDeadlockRetry(dao *DataAccessLayer, funcLbl string, query string) error {
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	retryOnErrorMessageContains := []string{"Duplicate entry", "Deadlock", "Lock wait timeout exceeded", sql.ErrNoRows.Error(), "First Attempt"}
	logger := dao.GetLogger()
	// Looper with forced entry (actual err for retries is assigned at bottom of the loop)
	err := fmt.Errorf("First Attempt")
	for retryCounter > 0 && err != nil && containsAny(err.Error(), retryOnErrorMessageContains) {
		if retryCounter != dao.DeadlockRetryCounter {
			logger.Debug("restarting transaction", zap.String("funcLbl", funcLbl), zap.String("retryReason", firstMatch(err.Error(), retryOnErrorMessageContains)), zap.Int64("retryCounter", retryCounter))
		}
		var errBeginx error
		tx, errBeginx := dao.MetadataDB.Beginx()
		if errBeginx != nil {
			return fmt.Errorf("%s could not begin transaction, %s", funcLbl, errBeginx.Error())
		}
		retryCounter--
		var errPreparex error
		stmt, errPreparex := tx.Preparex(query)
		if errPreparex != nil {
			tx.Rollback()
			return fmt.Errorf("%s error preparing key statement, %s", funcLbl, errPreparex.Error())
		}
		err = nil
		_, err = stmt.Exec()
		stmt.Close()
		if err != nil {
			tx.Rollback()
			time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		} else {
			tx.Commit()
		}
	}
	return nil
}

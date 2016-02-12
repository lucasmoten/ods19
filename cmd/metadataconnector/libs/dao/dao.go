package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// DAO defines the contract our app has with the database.
type DAO interface {
	// AddAddPermissionToObject(createdBy string, object *models.ODObject, permission *models.ODObjectPermission) error
	// AddPropertyToObject(createdBy string, object *models.ODObject, property *models.ODProperty) error
	// CreateObject(object *models.ODObject, acm *models.ODACM) error
	// CreateObjectType(objectType *models.ODObjectType) error
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
	// GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error)
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

// FakeDAO is suitable for tests. Add fields to this struct to hold fake
// reponses for each of the methods that FakeDAO will implement. These fake
// response fields can be explicitly set in tests.
type FakeDAO struct {
	Object            *models.ODObject
	ObjectPermissions []models.ODObjectPermission
	ObjectProperites  []models.ODObjectPropertyEx
	// TODO: all required responses should be fields.
}

// GetObject for FakeDAO.
func (fake *FakeDAO) GetObject(object *models.ODObject, loadProperties bool) (*models.ODObject, error) {
	// return what we set on the field
	return fake.Object, nil
}

// GetPermissionsForObject for FakeDAO.
func (fake *FakeDAO) GetPermissionsForObject(object *models.ODObject) ([]models.ODObjectPermission, error) {
	return fake.ObjectPermissions, nil
}

// GetPropertiesForObject for FakeDAO.
func (fake *FakeDAO) GetPropertiesForObject(object *models.ODObject) ([]models.ODObjectPropertyEx, error) {
	return fake.ObjectProperites, nil
}

// DataAccessLayer is a concrete DAO implementation with a true db conn.
type DataAccessLayer struct {
	MetadataDB *sqlx.DB
}

// GetObject uses the passed in object and makes the appropriate sql calls to
// the database to retrieve and return the requested object by ID. Optionally,
// loadProperties flag pulls in nested properties associated with the object.
func (dao *DataAccessLayer) GetObject(object *models.ODObject, loadProperties bool) (*models.ODObject, error) {
	var dbObject models.ODObject
	getObjectStatement := `select o.*, ot.name typeName from object o inner join object_type ot on o.typeid = ot.id where o.id = ?`
	err := dao.MetadataDB.Get(&dbObject, getObjectStatement, object.ID)
	if err != nil {
		return &dbObject, err
	}

	dbObject.Permissions, err = dao.GetPermissionsForObject(&dbObject)
	if err != nil {
		return &dbObject, err
	}

	// Load properties if requested
	if loadProperties {
		dbObject.Properties, err = dao.GetPropertiesForObject(&dbObject)
		if err != nil {
			return &dbObject, err
		}
	}

	// All ready ....
	return &dbObject, err
}

// GetPermissionsForObject retrieves the grants for a given object.
func (dao *DataAccessLayer) GetPermissionsForObject(object *models.ODObject) ([]models.ODObjectPermission, error) {

	response := []models.ODObjectPermission{}
	query := `select op.* from object_permission op inner join object o on op.objectid = o.id where op.isdeleted = 0 and op.objectid = ?`
	err := dao.MetadataDB.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

// GetPropertiesForObject retrieves the properties for a given object.
func (dao *DataAccessLayer) GetPropertiesForObject(object *models.ODObject) ([]models.ODObjectPropertyEx, error) {
	response := []models.ODObjectPropertyEx{}
	query := `select p.* from property p
            inner join object_property op on p.id = op.propertyid
            where p.isdeleted = 0 and op.isdeleted = 0 and op.objectid = ?`
	err := dao.MetadataDB.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

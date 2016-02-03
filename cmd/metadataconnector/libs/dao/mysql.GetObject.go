package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObject uses the passed in object and makes the appropriate sql calls to
// the database to retrieve and return the requested object by ID.  Optionally,
// loadProperties flag will pull in the nested properties associated with the
// object
func GetObject(db *sqlx.DB, object *models.ODObject, loadProperties bool) (*models.ODObject, error) {

	var dbObject models.ODObject
	getObjectStatement := `select o.*, ot.name typeName from object o inner join object_type ot on o.typeid = ot.id where o.id = ?`
	err := db.Get(&dbObject, getObjectStatement, object.ID)
	if err != nil {
		return &dbObject, err
	}

	// Load permissions always
	//XXX --- we load the permissions for *this* user for *this* object.
	// The object could have a very large number of permissions associated
	//with it, of which only one is relevant.
	dbObject.Permissions, err = GetPermissionsForObject(db, &dbObject)
	if err != nil {
		return &dbObject, err
	}

	// Load properties if requested
	if loadProperties {
		dbObject.Properties, err = GetPropertiesForObject(db, &dbObject)
		if err != nil {
			return &dbObject, err
		}
	}

	// All ready ....
	return &dbObject, err
}

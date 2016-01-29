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

	if loadProperties {
		dbObject.Properties, err = GetPropertiesForObject(db, &dbObject)
		if err != nil {
			return &dbObject, err
		}
	}

	return &dbObject, err
}

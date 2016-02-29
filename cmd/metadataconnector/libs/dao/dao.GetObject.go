package dao

import (
	"log"
	"strconv"
	"time"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObject uses the passed in object and makes the appropriate sql calls to
// the database to retrieve and return the requested object by ID. Optionally,
// loadProperties flag pulls in nested properties associated with the object.
func (dao *DataAccessLayer) GetObject(object models.ODObject, loadProperties bool) (models.ODObject, error) {

	tx := dao.MetadataDB.MustBegin()
	dbObject, err := getObjectInTransaction(tx, object, loadProperties)
	if err != nil {
		log.Printf("Error in GetObject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObject, err
}

func getObjectInTransaction(tx *sqlx.Tx, object models.ODObject, loadProperties bool) (models.ODObject, error) {
	var dbObject models.ODObject

	x := strconv.Itoa(time.Now().UTC().Nanosecond())

	getObjectStatement := `select o.*, ot.name typeName, '` + x + `' nanosecond from object o inner join object_type ot on o.typeid = ot.id where o.id = ?`
	err := tx.Unsafe().Get(&dbObject, getObjectStatement, object.ID)
	if err == nil {
		dbPermissions, dbPermErr := getPermissionsForObjectInTransaction(tx, object)
		dbObject.Permissions = dbPermissions
		if dbPermErr != nil {
			err = dbPermErr
		} else {
			// Load properties if requested
			if loadProperties {
				dbProperties, dbPropErr := getPropertiesForObjectInTransaction(tx, object)
				dbObject.Properties = dbProperties
				if dbPropErr != nil {
					err = dbPropErr
				}
			}
		}
	}
	return dbObject, err
}

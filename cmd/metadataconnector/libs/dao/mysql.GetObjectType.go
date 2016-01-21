package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectType uses the passed in objectType and makes the appropriate sql
// calls to the database to retrieve and return the requested object type by ID.
func GetObjectType(db *sqlx.DB, objectType *models.ODObjectType) (*models.ODObjectType, error) {
	var dbObjectType models.ODObjectType
	getObjectTypeStatement := `select * from object_type where id = ?`
	err := db.Get(&dbObjectType, getObjectTypeStatement, objectType.ID)
	if err != nil {
		return &dbObjectType, err
	}

	return &dbObjectType, err
}

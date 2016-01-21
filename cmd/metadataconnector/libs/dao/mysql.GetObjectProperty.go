package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectProperty uses the passed in object property and makes the
// appropriate sql calls to the database to retrieve and return the requested
// property by ID.
func GetObjectProperty(db *sqlx.DB, objectProperty *models.ODObjectPropertyEx) (*models.ODObjectPropertyEx, error) {
	var dbObjectProperty models.ODObjectPropertyEx
	query := `select * from property where id = ?`
	err := db.Get(&dbObjectProperty, query, objectProperty.ID)
	if err != nil {
		print(err.Error())
	}
	return &dbObjectProperty, err
}

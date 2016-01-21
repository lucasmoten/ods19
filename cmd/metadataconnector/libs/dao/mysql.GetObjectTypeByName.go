package dao

import (
	"database/sql"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectTypeByName looks up an object type by its name, and if it doesn't
// exist, optionally calls CreateObjectType to add it.
func GetObjectTypeByName(db *sqlx.DB, typeName string, addIfMissing bool, createdBy string) models.ODObjectType {
	var objectType models.ODObjectType
	// Get the ID of the newly created object and assign to passed in object
	getObjectTypeStatement := `select * from object_type where name = ?	and isdeleted = 0 order by createddate desc limit 1`
	err := db.Get(&objectType, getObjectTypeStatement, typeName)
	if err != nil {
		if err == sql.ErrNoRows {
			if addIfMissing {
				objectType.Name = typeName
				objectType.CreatedBy = createdBy
				CreateObjectType(db, &objectType)
			} // if addIfMissing {
		} else {
			panic(err)
		} // if err == sql.NoRows
	} // if err != nil

	return objectType
}

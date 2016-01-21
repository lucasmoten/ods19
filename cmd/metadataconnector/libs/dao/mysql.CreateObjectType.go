package dao

import (
	"database/sql"
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// CreateObjectType adds a new object type definition to the database based upon
// the passed in object type settings.  At a minimm, createdBy and the name of
// the object type must exist.  Once added, the record is retrieved and the
// object type passed in by reference is updated with the remaining attributes
func CreateObjectType(db *sqlx.DB, objectType *models.ODObjectType) {
	// Setup the statement
	addObjectTypeStatement, err := db.Prepare(`insert object_type set createdBy = ?, name = ?, description = ?, contentConnector = ?`)
	if err != nil {
		print(err.Error())
	}
	// Add it
	result, err := addObjectTypeStatement.Exec(objectType.CreatedBy, objectType.Name, objectType.Description.String, objectType.ContentConnector.String)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if rowCount < 1 {
		fmt.Println("No rows added from inserting object type")
		return
	}
	// Get the ID of the newly created object type and assign to passed in objectType
	getObjectTypeStatement := `select * from object_type where createdBy = ? and name = ? and isdeleted = 0 order by createdDate desc limit 1`
	err = db.Get(objectType, getObjectTypeStatement, objectType.CreatedBy, objectType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Type was not found even after just adding!")
		}
	}
}

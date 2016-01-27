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
func CreateObjectType(db *sqlx.DB, objectType *models.ODObjectType) error {
	// Setup the statement
	addObjectTypeStatement, err := db.Prepare(`insert object_type set createdBy = ?, name = ?, description = ?, contentConnector = ?`)
	if err != nil {
		fmt.Println("error 19")
		return fmt.Errorf("CreateObjectType error preparing add object type statement, %s", err.Error())
	}
	// Add it
	result, err := addObjectTypeStatement.Exec(objectType.CreatedBy, objectType.Name, objectType.Description.String, objectType.ContentConnector.String)
	if err != nil {
		fmt.Println("error 25")
		return fmt.Errorf("CreateObjectType error executing add object type statement, %s", err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		fmt.Println("error 31")
		return fmt.Errorf("CreateObjectType error checking rows affected, %s", err.Error())
	}
	if rowCount < 1 {
		fmt.Println("error 35")
		return fmt.Errorf("CreateObjectType there was less than one row affected")
	}
	// Get the ID of the newly created object type and assign to passed in objectType
	getObjectTypeStatement := `select * from object_type where createdBy = ? and name = ? and isdeleted = 0 order by createdDate desc limit 1`
	err = db.Get(objectType, getObjectTypeStatement, objectType.CreatedBy, objectType.Name)
	if err != nil {
		fmt.Println("error 42")
		if err == sql.ErrNoRows {
			fmt.Println("error 44")
			return fmt.Errorf("CreateObjectType type was not found even after just adding it!, %s", err.Error())
		}
		fmt.Println("error 47")
		return fmt.Errorf("CreateObjectType error getting newly added object type, %s", err.Error())
	}
	return nil
}

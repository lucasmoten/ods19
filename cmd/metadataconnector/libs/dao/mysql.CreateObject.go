package dao

import (
	"fmt"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// CreateObject uses the passed in object and acm configuration and makes the
// appropriate sql calls to the database to insert the object, insert the acm
// configuration, associate the two together. Identifiers are captured and
// assigned to the relevant objects
func CreateObject(db *sqlx.DB, object *models.ODObject, acm *models.ODACM) error {

	// lookup type, assign its id to the object for reference
	if object.TypeID == nil {
		objectType, err := GetObjectTypeByName(db, object.TypeName.String, true, object.CreatedBy)
		if err != nil {
			fmt.Println("error 20")
			return fmt.Errorf("CreateObject Error calling GetObjectTypeByName, %s", err.Error())
		}
		object.TypeID = objectType.ID
	}

	// insert object
	addObjectStatement, err := db.Prepare(`insert object set createdBy = ?, typeId = ?, name = ?, description = ?, parentId = ?, contentConnector = ?, encryptIV = ?, encryptKey = ?, contentType = ?, contentSize = ? `)
	if err != nil {
		fmt.Println("error 29")
		return fmt.Errorf("CreateObject Preparing add object statement, %s", err.Error())
	}
	// Add it
	result, err := addObjectStatement.Exec(object.CreatedBy, object.TypeID,
		object.Name, object.Description.String, object.ParentID,
		object.ContentConnector.String, object.EncryptIV.String,
		object.EncryptKey.String, object.ContentType.String,
		object.ContentSize)
	if err != nil {
		fmt.Println("error 39")
		return fmt.Errorf("CreateObject Error executing add object statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Println("error 44")
		return fmt.Errorf("CreateObject Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		fmt.Println("error 48")
		return fmt.Errorf("CreateObject object inserted but no rows affected!")
	}
	// Get the ID of the newly created object and assign to passed in object
	// The following block uses all parameters but doesnt take into account null
	// values...
	// getObjectStatement := `select * from object where createdby = ? and typeId = ? and name = ? and description = ? and parentId = ? and contentConnector = ? and encryptIV = ? and encryptKey = ? and contentType = ? and contentSize = ? and isdeleted = 0 order by createddate desc limit 1`
	// err = db.Get(object, getObjectStatement, object.CreatedBy, object.TypeID,
	// 	object.Name, object.Description.String, object.ParentID,
	// 	object.ContentConnector.String, object.EncryptIV.String,
	// 	object.EncryptKey.String, object.ContentType.String,
	// 	object.ContentSize)
	getObjectStatement := `select * from object where createdby = ? and typeId = ? and name = ? and isdeleted = 0 order by createddate desc limit 1`
	err = db.Get(object, getObjectStatement, object.CreatedBy, object.TypeID, object.Name)
	if err != nil {
		fmt.Println("error 63")
		return fmt.Errorf("CreateObject Error retrieving object, %s", err.Error())
	}

	// TODO: add properties of object.Properties []models.ODObjectPropertyEx

	// insert acm

	return nil
}
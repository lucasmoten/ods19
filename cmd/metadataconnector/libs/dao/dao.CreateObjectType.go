package dao

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// CreateObjectType adds a new object type definition to the database based upon
// the passed in object type settings.  At a minimm, createdBy and the name of
// the object type must exist.  Once added, the record is retrieved and the
// object type passed in by reference is updated with the remaining attributes
func (dao *DataAccessLayer) CreateObjectType(objectType *models.ODObjectType) error {
	tx := dao.MetadataDB.MustBegin()
	err := createObjectTypeInTransaction(tx, objectType)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func createObjectTypeInTransaction(tx *sqlx.Tx, objectType *models.ODObjectType) error {

	addObjectTypeStatement, err := tx.Prepare(
		`insert object_type set createdBy = ?, name = ?, description = ?, contentConnector = ?`)
	if err != nil {
		return fmt.Errorf("CreateObjectType error preparing add object type statement, %s", err.Error())
	}
	// Add it
	result, err := addObjectTypeStatement.Exec(objectType.CreatedBy, objectType.Name, objectType.Description.String, objectType.ContentConnector.String)
	if err != nil {
		return fmt.Errorf("CreateObjectType error executing add object type statement, %s", err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("CreateObjectType error checking rows affected, %s", err.Error())
	}
	if rowCount < 1 {
		return fmt.Errorf("CreateObjectType there was less than one row affected")
	}
	// Get the ID of the newly created object type and assign to passed in objectType
	getObjectTypeStatement := `select * from object_type where createdBy = ?
  and name = ? and isdeleted = 0 order by createdDate desc limit 1`
	err = tx.Get(objectType, getObjectTypeStatement, objectType.CreatedBy, objectType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("CreateObjectType type was not found even after just adding it!, %s", err.Error())
		}
		return fmt.Errorf("CreateObjectType error getting newly added object type, %s", err.Error())
	}
	return nil
}

package dao

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
)

// CreateObjectType adds a new object type definition to the database based upon
// the passed in object type settings.  At a minimm, createdBy and the name of
// the object type must exist.  Once added, the record is retrieved and the
// object type passed in by reference is updated with the remaining attributes
func (dao *DataAccessLayer) CreateObjectType(objectType *models.ODObjectType) (models.ODObjectType, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		log.Printf("Could not begin transaction: %v", err)
		return models.ODObjectType{}, err
	}
	dbObjectType, err := createObjectTypeInTransaction(tx, objectType)
	if err != nil {
		log.Printf("Error in CreateObjectType: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectType, err
}

func createObjectTypeInTransaction(tx *sqlx.Tx, objectType *models.ODObjectType) (models.ODObjectType, error) {
	var dbObjectType models.ODObjectType
	addObjectTypeStatement, err := tx.Preparex(`insert object_type set 
        createdBy = ?
        ,name = ?
        ,description = ?
        ,contentConnector = ?
    `)
	if err != nil {
		return dbObjectType, fmt.Errorf("CreateObjectType error preparing add object type statement, %s", err.Error())
	}
	// Add it
	result, err := addObjectTypeStatement.Exec(objectType.CreatedBy, objectType.Name, objectType.Description.String, objectType.ContentConnector.String)
	if err != nil {
		return dbObjectType, fmt.Errorf("CreateObjectType error executing add object type statement, %s", err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		return dbObjectType, fmt.Errorf("CreateObjectType error checking rows affected, %s", err.Error())
	}
	if rowCount < 1 {
		return dbObjectType, fmt.Errorf("CreateObjectType there was less than one row affected")
	}
	// Get the ID of the newly created object type and assign to passed in objectType
	getObjectTypeStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector
    from object_type 
    where 
        createdBy = ?
        and name = ? 
        and isdeleted = 0 
    order by createdDate desc limit 1`
	err = tx.Get(&dbObjectType, getObjectTypeStatement, objectType.CreatedBy, objectType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbObjectType, fmt.Errorf("CreateObjectType type was not found even after just adding it!, %s", err.Error())
		}
		return dbObjectType, fmt.Errorf("CreateObjectType error getting newly added object type, %s", err.Error())
	}
	return dbObjectType, nil
}

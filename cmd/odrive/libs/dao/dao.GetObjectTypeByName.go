package dao

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
)

// GetObjectTypeByName looks up an object type by its name, and if it doesn't
// exist, optionally calls CreateObjectType to add it.
func (dao *DataAccessLayer) GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		log.Printf("Could not begin transaction: %v", err)
		return models.ODObjectType{}, err
	}
	objectType, err := getObjectTypeByNameInTransaction(tx, typeName, addIfMissing, createdBy)
	if err != nil {
		log.Printf("Error in GetObjectTypeByName: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return objectType, err
}

func getObjectTypeByNameInTransaction(tx *sqlx.Tx, typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {

	var objectType models.ODObjectType
	// Get the ID of the newly created object and assign to passed in object
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
        ,ownedBy
        ,name
        ,description
        ,contentConnector
    from
        object_type
    where
        name = ?
    order by isDeleted asc, createdDate desc limit 1    
    `
	err := tx.Get(&objectType, getObjectTypeStatement, typeName)
	if err != nil {
		if err == sql.ErrNoRows {
			if addIfMissing {
				// Clear the error from no rows
				err = nil
				// Prepare new object type
				objectType.Name = typeName
				objectType.CreatedBy = createdBy
				// Create it
				createdObjectType, err := createObjectTypeInTransaction(tx, &objectType)
				// Any errors? return them
				if err != nil {
					// Empty return with error from creation
					return objectType, fmt.Errorf("Error creating object when missing: %s", err.Error())
				}
				// Assign created type to the return value
				objectType = createdObjectType
			}
		} else {
			// Some other error besides no matching rows...
			// Empty return type with error retrieving
			return objectType, fmt.Errorf("GetObjectTypeByName error, %s", err.Error())
		}
	}

	// Need to make sure the type retrieved isn't deleted.
	if objectType.IsDeleted {
		// Existing type is deleted. Should a new active type be created?
		if addIfMissing {
			// Prepare new object type
			objectType.Name = typeName
			objectType.CreatedBy = createdBy
			// Create it
			createdObjectType, err := createObjectTypeInTransaction(tx, &objectType)
			// Any errors? return them
			if err != nil {
				// Reinitialize objectType first since it may be dirty at this
				// phase and caller may accidentally use if not properly
				// checking errors
				objectType = models.ODObjectType{}
				return objectType, fmt.Errorf("Error recreating object when previous was deleted: %s", err.Error())
			}
			// Assign created type to the return value
			objectType = createdObjectType
		}
	}

	// Return response
	return objectType, err
}

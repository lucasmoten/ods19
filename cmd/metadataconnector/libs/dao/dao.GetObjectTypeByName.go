package dao

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// GetObjectTypeByName looks up an object type by its name, and if it doesn't
// exist, optionally calls CreateObjectType to add it.
func (dao *DataAccessLayer) GetObjectTypeByName(typeName string, addIfMissing bool, createdBy string) (models.ODObjectType, error) {
	tx := dao.MetadataDB.MustBegin()
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
	getObjectTypeStatement := `select * from object_type where name = ? order by isdeleted asc, createddate desc limit 1`
	err := tx.Get(&objectType, getObjectTypeStatement, typeName)
	if err != nil {
		if err == sql.ErrNoRows {
			if addIfMissing {
				objectType.Name = typeName
				objectType.CreatedBy = createdBy
				err = createObjectTypeInTransaction(tx, &objectType)
			}
		} else {
			return objectType, fmt.Errorf("GetObjectTypeByName error, %s", err.Error())
		}
	}
	if objectType.IsDeleted && addIfMissing {
		objectType.Name = typeName
		objectType.CreatedBy = createdBy
		err = createObjectTypeInTransaction(tx, &objectType)
	}

	return objectType, err
}

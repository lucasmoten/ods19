package dao

import (
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectType uses the passed in objectType and makes the appropriate sql
// calls to the database to retrieve and return the requested object type by ID.
func (dao *DataAccessLayer) GetObjectType(objectType models.ODObjectType) (*models.ODObjectType, error) {
	tx := dao.MetadataDB.MustBegin()
	dbObjectType, err := getObjectTypeInTransaction(tx, objectType)
	if err != nil {
		log.Printf("Error in GetObjectType: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectType, err
}

func getObjectTypeInTransaction(tx *sqlx.Tx, objectType models.ODObjectType) (*models.ODObjectType, error) {
	var dbObjectType models.ODObjectType
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
    from
        object_type
    where
        id = ?    
    `
	err := tx.Get(&dbObjectType, getObjectTypeStatement, objectType.ID)
	if err != nil {
		return &dbObjectType, err
	}

	return &dbObjectType, err
}

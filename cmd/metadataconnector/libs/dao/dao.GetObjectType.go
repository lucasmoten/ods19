package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectType uses the passed in objectType and makes the appropriate sql
// calls to the database to retrieve and return the requested object type by ID.
func (dao *DataAccessLayer) GetObjectType(objectType *models.ODObjectType) (*models.ODObjectType, error) {
	tx := dao.MetadataDB.MustBegin()
	dbObjectType, err := getObjectTypeInTransaction(tx, objectType)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectType, err
}

func getObjectTypeInTransaction(tx *sqlx.Tx, objectType *models.ODObjectType) (*models.ODObjectType, error) {
	var dbObjectType models.ODObjectType
	getObjectTypeStatement := `select * from object_type where id = ?`
	err := tx.Get(&dbObjectType, getObjectTypeStatement, objectType.ID)
	if err != nil {
		return &dbObjectType, err
	}

	return &dbObjectType, err
}

package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectProperty return the requested property by ID.
// NOTE: Should we just pass an ID instead?
func (dao *DataAccessLayer) GetObjectProperty(objectProperty *models.ODObjectPropertyEx) (*models.ODObjectPropertyEx, error) {
	tx := dao.MetadataDB.MustBegin()
	dbObjectProperty, err := getObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectProperty, err
}

func getObjectPropertyInTransaction(tx *sqlx.Tx, objectProperty *models.ODObjectPropertyEx) (*models.ODObjectPropertyEx, error) {
	var dbObjectProperty models.ODObjectPropertyEx
	query := `select * from property where id = ?`
	err := tx.Get(&dbObjectProperty, query, objectProperty.ID)
	if err != nil {
		print(err.Error())
	}
	return &dbObjectProperty, err
}

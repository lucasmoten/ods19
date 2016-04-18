package dao

import (
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetObjectPermission return the requested permission by ID.
// NOTE: Should we just pass an ID instead?
func (dao *DataAccessLayer) GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
	tx := dao.MetadataDB.MustBegin()
	dbObjectPermission, err := getObjectPermissionInTransaction(tx, objectPermission)
	if err != nil {
		log.Printf("Error in GetObjectPermission: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectPermission, err
}

func getObjectPermissionInTransaction(tx *sqlx.Tx, objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
	var dbObjectPermission models.ODObjectPermission
	query := `select * from object_permission where id = ?`
	err := tx.Get(&dbObjectPermission, query, objectPermission.ID)
	if err != nil {
		print(err.Error())
	}
	return dbObjectPermission, err
}

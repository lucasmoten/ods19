package dao

import (
	"errors"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// AddPermissionToObject creates a new permission with the provided object id,
// grant, and permissions.
func (dao *DataAccessLayer) AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := addPermissionToObjectInTransaction(tx, object, permission)
	if err != nil {
		log.Printf("Error in AddPermissionToobject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func addPermissionToObjectInTransaction(tx *sqlx.Tx, object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error) {

	var dbPermission models.ODObjectPermission

	// Setup the statement
	addPermissionStatement, err := tx.Preparex(`insert object_permission set createdby = ?, objectId = ?, grantee = ?, allowCreate = ?, allowRead = ?, allowUpdate = ?, allowDelete = ?, encryptKey = ?`)
	if err != nil {
		return dbPermission, err
	}
	// Add it
	result, err := addPermissionStatement.Exec(permission.CreatedBy, object.ID, permission.Grantee, permission.AllowCreate, permission.AllowRead, permission.AllowUpdate, permission.AllowDelete, permission.EncryptKey)
	if err != nil {
		return dbPermission, err
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		return dbPermission, errors.New("No rows added from inserting permission")
	}
	addPermissionStatement.Close()
	// Get the ID of the newly created permission
	var newPermissionID []byte
	getPermissionIDStatement, err := tx.Preparex(`select id from object_permission where createdby = ? and objectId = ? and grantee = ? and isdeleted = 0 order by createddate desc limit 1`)
	if err != nil {
		return dbPermission, err
	}
	err = getPermissionIDStatement.QueryRowx(permission.CreatedBy, object.ID, permission.Grantee).Scan(&newPermissionID)
	if err != nil {
		return dbPermission, err
	}
	getPermissionIDStatement.Close()
	// Retrieve back into permission
	err = tx.Get(&dbPermission, `select * from object_permission where id = ?`, newPermissionID)
	if err != nil {
		return dbPermission, err
	}
	*permission = dbPermission
	//permission = &dbPermission

	return dbPermission, nil
}

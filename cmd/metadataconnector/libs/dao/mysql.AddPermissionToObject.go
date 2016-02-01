package dao

import (
	"errors"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// AddPermissionToObject creates a new permission with the provided object id,
// grant, and permissions
func AddPermissionToObject(db *sqlx.DB, createdBy string, object *models.ODObject, permission *models.ODObjectPermission) error {
	tx := db.MustBegin()
	// Setup the statement
	addPermissionStatement, err := tx.Prepare(`insert object_permission set createdby = ?, objectId = ?, grantee = ?, allowCreate = ?, allowRead = ?, allowUpdate = ?, allowDelete = ?, encryptKey = ?`)
	if err != nil {
		return err
	}
	// Add it
	result, err := addPermissionStatement.Exec(createdBy, object.ID, permission.Grantee, permission.AllowCreate, permission.AllowRead, permission.AllowUpdate, permission.AllowDelete, permission.EncryptKey)
	if err != nil {
		return err
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		return errors.New("No rows added from inserting permission")
	}
	// Get the ID of the newly created permission
	var newPermissionID []byte
	getPermissionIDStatement, err := tx.Prepare(`select id from object_permission where createdby = ? and objectId = ? and grantee = ? and isdeleted = 0 order by createddate desc limit 1`)
	if err != nil {
		return err
	}
	err = getPermissionIDStatement.QueryRow(createdBy, object.ID, permission.Grantee).Scan(&newPermissionID)
	if err != nil {
		return err
	}
	// Retrieve back into permission
	err = tx.Get(permission, `select * from object_permission where id = ?`, newPermissionID)
	if err != nil {
		return err
	}
	tx.Commit()

	return nil
}

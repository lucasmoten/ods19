package dao

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
)

// UpdatePermission uses the passed in permission and makes the appropriate
// sql calls to the database to update the existing grant
func (dao *DataAccessLayer) UpdatePermission(permission models.ODObjectPermission) error {
	tx := dao.MetadataDB.MustBegin()
	err := updatePermissionInTransaction(tx, permission)
	if err != nil {
		log.Printf("Error in UpdateObjectPermission: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func updatePermissionInTransaction(tx *sqlx.Tx, permission models.ODObjectPermission) error {

	updatePermissionStatement, err := tx.Preparex(`update object_permission set modifiedBy = ?, grantee = ?, allowCreate = ?, allowRead = ?, allowUpdate = ?, allowDelete = ?, encryptKey = ? where id = ? and changeToken = ?`)
	if err != nil {
		return fmt.Errorf("UpdatePermission Preparing update statement, %s", err.Error())
	}
	result, err := updatePermissionStatement.Exec(permission.ModifiedBy,
		permission.Grantee, permission.AllowCreate, permission.AllowRead,
		permission.AllowUpdate, permission.AllowDelete, permission.EncryptKey,
		permission.ID, permission.ChangeToken)
	if err != nil {
		return fmt.Errorf("UpdatePermission Error executing update statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("UpdatePermission Error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return fmt.Errorf("UpdatePermission Did not affect any rows (Possible bad ID or changeToken)!")
	}

	return nil
}

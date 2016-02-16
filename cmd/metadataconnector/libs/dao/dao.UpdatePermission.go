package dao

import (
	"fmt"

	"decipher.com/oduploader/metadata/models"
)

// UpdatePermission uses the passed in permission and makes the appropriate
// sql calls to the database to update the existing grant
func (dao *DataAccessLayer) UpdatePermission(permission *models.ODObjectPermission) error {

	updatePermissionStatement, err := dao.MetadataDB.Prepare(`update object_permission set modifiedBy = ?, grantee = ?, allowCreate = ?, allowRead = ?, allowUpdate = ?, allowDelete = ?, encryptKey = ? where id = ? and changeToken = ?`)
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

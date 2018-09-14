package dao

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// UpdatePermission uses the passed in permission and makes the appropriate
// sql calls to the database to update the existing grant
func (dao *DataAccessLayer) UpdatePermission(permission models.ODObjectPermission) error {
	defer util.Time("UpdatePermission")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return err
	}
	err = updatePermissionInTransaction(tx, permission)
	if err != nil {
		dao.GetLogger().Error("Error in UpdatePermission", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func updatePermissionInTransaction(tx *sqlx.Tx, permission models.ODObjectPermission) error {

	updatePermissionStatement, err := tx.Preparex(`update object_permission set 
        modifiedBy = ?
        ,encryptKey = ? 
		,permissionIV = ?
		,permissionMAC = ?
	where id = ? and changeToken = ?`)
	defer updatePermissionStatement.Close()
	if err != nil {
		return fmt.Errorf("updatePermission error preparing update statement, %s", err.Error())
	}
	result, err := updatePermissionStatement.Exec(permission.ModifiedBy,
		permission.EncryptKey, permission.PermissionIV, permission.PermissionMAC,
		permission.ID, permission.ChangeToken)
	if err != nil {
		return fmt.Errorf("updatePermission error executing update statement, %s", err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updatePermission error checking result for rows affected, %s", err.Error())
	}
	if rowsAffected <= 0 {
		return fmt.Errorf("updatePermission did not affect any rows, possibly bad id or changetoken")
	}

	return nil
}

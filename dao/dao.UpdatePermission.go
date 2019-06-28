package dao

import (
	"encoding/hex"
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
	logger := dao.GetLogger()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		logger.Error("Could not begin transaction", zap.Error(err))
		return err
	}
	err = updatePermissionInTransaction(logger, tx, permission)
	if err != nil {
		logger.Error("Error in UpdatePermission", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func updatePermissionInTransaction(logger *zap.Logger, tx *sqlx.Tx, permission models.ODObjectPermission) error {

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
		// DIMEODS-1262 - log this, but don't return as failure
		logger.Debug("updatePermission did not affect any rows, possibly bad id or changetoken", zap.String("id", hex.EncodeToString(permission.ID)), zap.String("changetoken", permission.ChangeToken))
	}

	return nil
}

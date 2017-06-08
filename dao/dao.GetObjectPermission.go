package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetObjectPermission return the requested permission by ID.
// NOTE: Should we just pass an ID instead?
func (dao *DataAccessLayer) GetObjectPermission(objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
	defer util.Time("GetObjectPermission")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectPermission{}, err
	}
	dbObjectPermission, err := getObjectPermissionInTransaction(tx, objectPermission)
	if err != nil {
		dao.GetLogger().Error("Error in GetObjectPermission", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbObjectPermission, err
}

func getObjectPermissionInTransaction(tx *sqlx.Tx, objectPermission models.ODObjectPermission) (models.ODObjectPermission, error) {
	var dbObjectPermission models.ODObjectPermission
	query := `
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
        ,objectId
        ,grantee
        ,allowCreate
        ,allowRead
        ,allowUpdate
        ,allowDelete
        ,allowShare
        ,explicitShare
        ,encryptKey     
		,permissionIV
		,permissionMAC
    from object_permission 
    where id = ?`
	err := tx.Get(&dbObjectPermission, query, objectPermission.ID)
	if err != nil {
		print(err.Error())
	}
	return dbObjectPermission, err
}

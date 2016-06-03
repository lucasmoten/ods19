package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetPermissionsForObject retrieves the grants for a given object.
func (dao *DataAccessLayer) GetPermissionsForObject(object models.ODObject) ([]models.ODObjectPermission, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return []models.ODObjectPermission{}, err
	}
	response, err := getPermissionsForObjectInTransaction(tx, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetPermissionsForObject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err

}

func getPermissionsForObjectInTransaction(tx *sqlx.Tx, object models.ODObject) ([]models.ODObjectPermission, error) {
	response := []models.ODObjectPermission{}
	query := `
    select 
        op.id
        ,op.createdDate
        ,op.createdBy
        ,op.modifiedDate
        ,op.modifiedBy
        ,op.isDeleted
        ,op.deletedDate
        ,op.deletedBy
        ,op.changeCount
        ,op.changeToken
        ,op.objectId
        ,op.grantee
        ,op.allowCreate
        ,op.allowRead
        ,op.allowUpdate
        ,op.allowDelete
        ,op.allowShare
        ,op.explicitShare
        ,op.encryptKey
    from object_permission op 
        inner join object o on op.objectid = o.id 
    where 
        op.isdeleted = 0 
        and op.objectid = ?`
	err := tx.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	return response, err
}

package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetPermissionsForObject retrieves the grants for a given object.
func (dao *DataAccessLayer) GetPermissionsForObject(object models.ODObject) ([]models.ODObjectPermission, error) {
	defer util.Time("GetPermissionsForObject")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return []models.ODObjectPermission{}, err
	}
	response, err := getPermissionsForObjectInTransaction(dao, tx, object)
	if err != nil {
		dao.GetLogger().Error("Error in GetPermissionsForObject", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err

}

func getPermissionsForObjectInTransaction(dao *DataAccessLayer, tx *sqlx.Tx, object models.ODObject) ([]models.ODObjectPermission, error) {
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
        ,op.acmShare
        ,op.allowCreate
        ,op.allowRead
        ,op.allowUpdate
        ,op.allowDelete
        ,op.allowShare
        ,op.explicitShare
        ,op.encryptKey
        ,op.permissionIV
        ,op.permissionMAC
    from 
		object_permission op 
    where 
        op.isdeleted = 0 
		and op.objectid = ?
	order by
		op.grantee`
	err := tx.Select(&response, query, object.ID)
	if err != nil {
		return response, err
	}
	for i, p := range response {
		response[i].AcmGrantee, err = getAcmGranteeInTransaction(dao, tx, p.Grantee)
		if err != nil {
			return response, err
		}
	}
	return response, err
}

package dao

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// AddPermissionToObject creates a new permission with the provided object id,
// grant, and permissions.
func (dao *DataAccessLayer) AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error) {
	defer util.Time("AddPermissionToObject")
	dao.GetLogger().Debug("dao starting txn for AddPermissionToObject")
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.Error(err))
		return models.ODObjectPermission{}, err
	}
	dao.GetLogger().Debug("dao passing  txn into addPermissionToObjectInTransaction")
	response, err := addPermissionToObjectInTransaction(tx, dao, object, permission)
	dao.GetLogger().Debug("dao returned txn from addPermissionToObjectInTransaction")
	if err != nil {
		dao.GetLogger().Error("error in addpermissiontoobject", zap.Error(err))
		dao.GetLogger().Debug("dao rolling back txn for AddPermissionToObject")
		tx.Rollback()
	} else {
		dao.GetLogger().Debug("dao committing txn for AddPermissionToObject")
		tx.Commit()
	}
	dao.GetLogger().Debug("dao finished txn for AddPermissionToObject")
	return response, err
}

func addPermissionToObjectInTransaction(tx *sqlx.Tx, dao *DataAccessLayer, object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error) {

	var dbPermission models.ODObjectPermission

	// Check that grantee specified exists
	permission.Grantee = models.AACFlatten(permission.Grantee)
	dao.GetLogger().Debug("dao passing  txn into getAcmGranteeInTransaction")
	dbAcmGrantee, dbAcmGranteeErr := getAcmGranteeInTransaction(tx, permission.Grantee)
	dao.GetLogger().Debug("dao returned txn from getAcmGranteeInTransaction")
	if dbAcmGranteeErr == sql.ErrNoRows {
		// Add if it didn't
		dao.GetLogger().Debug("dao passing  txn into createAcmGranteeInTransaction")
		dbAcmGrantee, dbAcmGranteeErr = createAcmGranteeInTransaction(tx, dao, permission.AcmGrantee)
		dao.GetLogger().Debug("dao returned txn from createAcmGranteeInTransaction")
		if dbAcmGranteeErr != nil {
			return dbPermission, dbAcmGranteeErr
		}
	}

	// Setup the statement
	addPermissionStatement, err := tx.Preparex(`insert object_permission set 
        createdby = ?
        ,objectId = ?
        ,grantee = ?
        ,acmShare = ?
        ,allowCreate = ?
        ,allowRead = ?
        ,allowUpdate = ?
        ,allowDelete = ?
        ,allowShare = ?
        ,explicitShare = ?
        ,encryptKey = ?
		,permissionIV = ?
		,permissionMAC = ?
    `)
	if err != nil {
		return dbPermission, err
	}
	defer addPermissionStatement.Close()
	// Add it
	result, err := addPermissionStatement.Exec(permission.CreatedBy, object.ID,
		dbAcmGrantee.Grantee, permission.AcmShare, permission.AllowCreate,
		permission.AllowRead, permission.AllowUpdate, permission.AllowDelete,
		permission.AllowShare, permission.ExplicitShare, permission.EncryptKey,
		permission.PermissionIV, permission.PermissionMAC)
	if err != nil {
		return dbPermission, err
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		return dbPermission, errors.New("No rows added from inserting permission")
	}
	// Get the ID of the newly created permission
	var newPermissionID []byte
	getPermissionIDStatement, err := tx.Preparex(`
    select 
        id 
    from object_permission 
    where 
        createdby = ? 
        and objectId = ? 
        and grantee = ? 
        and acmShare = ?
        and isdeleted = 0 
        and allowCreate = ? 
        and allowRead = ? 
        and allowUpdate = ? 
        and allowDelete = ? 
        and allowShare = ? 
    order by createddate desc limit 1
	`)
	if err != nil {
		return dbPermission, err
	}
	defer getPermissionIDStatement.Close()
	err = getPermissionIDStatement.QueryRowx(permission.CreatedBy, object.ID,
		permission.Grantee, permission.AcmShare, permission.AllowCreate, permission.AllowRead,
		permission.AllowUpdate, permission.AllowDelete,
		permission.AllowShare).Scan(&newPermissionID)
	if err != nil {
		return dbPermission, err
	}
	// Retrieve back into permission
	err = tx.Get(&dbPermission, `
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
        ,acmShare
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
    where id = ?
    `, newPermissionID)
	if err != nil {
		return dbPermission, err
	}
	*permission = dbPermission

	return dbPermission, nil
}

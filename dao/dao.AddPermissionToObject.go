package dao

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
)

// AddPermissionToObject creates a new permission with the provided object id,
// grant, and permissions.
func (dao *DataAccessLayer) AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error) {
	defer util.Time("AddPermissionToObject")
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("could not begin transaction", zap.String("err", err.Error()))
		return models.ODObjectPermission{}, err
	}
	response, err := addPermissionToObjectInTransaction(dao.GetLogger(), tx, object, permission)
	if err != nil {
		dao.GetLogger().Error("error in addpermissiontoobject", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func addPermissionToObjectInTransaction(logger *zap.Logger, tx *sqlx.Tx, object models.ODObject, permission *models.ODObjectPermission) (models.ODObjectPermission, error) {

	var dbPermission models.ODObjectPermission

	// Check that grantee specified exists
	permission.Grantee = models.AACFlatten(permission.Grantee)
	dbAcmGrantee, dbAcmGranteeErr := getAcmGranteeInTransaction(tx, permission.Grantee)
	if dbAcmGranteeErr == sql.ErrNoRows {
		// Add if it didnt
		dbAcmGrantee, dbAcmGranteeErr = createAcmGranteeInTransaction(logger, tx, permission.AcmGrantee)
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
	addPermissionStatement.Close()
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
	err = getPermissionIDStatement.QueryRowx(permission.CreatedBy, object.ID,
		permission.Grantee, permission.AcmShare, permission.AllowCreate, permission.AllowRead,
		permission.AllowUpdate, permission.AllowDelete,
		permission.AllowShare).Scan(&newPermissionID)
	if err != nil {
		return dbPermission, err
	}
	getPermissionIDStatement.Close()
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

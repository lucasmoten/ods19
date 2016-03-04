package dao

import (
	"errors"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
)

// AddPermissionToObject creates a new permission with the provided object id,
// grant, and permissions.
func (dao *DataAccessLayer) AddPermissionToObject(object models.ODObject, permission *models.ODObjectPermission, propogateToChildren bool, masterKey string) (models.ODObjectPermission, error) {
	tx := dao.MetadataDB.MustBegin()
	response, err := addPermissionToObjectInTransaction(tx, object, permission, propogateToChildren, masterKey)
	if err != nil {
		log.Printf("Error in AddPermissionToobject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return response, err
}

func addPermissionToObjectInTransaction(tx *sqlx.Tx, object models.ODObject, permission *models.ODObjectPermission, propagateToChildren bool, masterKey string) (models.ODObjectPermission, error) {

	var dbPermission models.ODObjectPermission

	// Setup the statement
	addPermissionStatement, err := tx.Preparex(`
    insert object_permission set createdby = ?, objectId = ?, grantee = ?, 
    allowCreate = ?, allowRead = ?, allowUpdate = ?, allowDelete = ?, 
    allowShare = ?,  explicitShare = ?, encryptKey = ?`)
	if err != nil {
		return dbPermission, err
	}
	// Add it
	result, err := addPermissionStatement.Exec(permission.CreatedBy, object.ID,
		permission.Grantee, permission.AllowCreate, permission.AllowRead,
		permission.AllowUpdate, permission.AllowDelete, permission.AllowShare,
		permission.ExplicitShare, permission.EncryptKey)
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
    select id from object_permission where createdby = ? and objectId = ? 
    and grantee = ? and isdeleted = 0 and allowCreate = ? and allowRead = ? 
    and allowUpdate = ? and allowDelete = ? and allowShare = ? 
    order by createddate desc limit 1`)
	if err != nil {
		return dbPermission, err
	}
	err = getPermissionIDStatement.QueryRowx(permission.CreatedBy, object.ID,
		permission.Grantee, permission.AllowCreate, permission.AllowRead,
		permission.AllowUpdate, permission.AllowDelete,
		permission.AllowShare).Scan(&newPermissionID)
	if err != nil {
		return dbPermission, err
	}
	getPermissionIDStatement.Close()
	// Retrieve back into permission
	err = tx.Get(&dbPermission, `select * from object_permission where id = ?`, newPermissionID)
	if err != nil {
		return dbPermission, err
	}
	*permission = dbPermission

	// Handle propagation to existing children
	if propagateToChildren {
		children, err := getChildObjectsInTransaction(tx, "", 1, MaxPageSize, object)
		if err != nil {
			return dbPermission, err
		}
		for _, childObject := range children.Objects {
			propagatedPermission := models.ODObjectPermission{}
			propagatedPermission.CreatedBy = permission.CreatedBy
			// - Same Grantee
			propagatedPermission.Grantee = permission.Grantee
			// - Propogated permissions are not explicit
			propagatedPermission.ExplicitShare = false
			// - Same permissions
			propagatedPermission.AllowCreate = permission.AllowCreate
			propagatedPermission.AllowRead = permission.AllowRead
			propagatedPermission.AllowUpdate = permission.AllowUpdate
			propagatedPermission.AllowDelete = permission.AllowDelete
			propagatedPermission.AllowShare = permission.AllowShare
			// - Encryption
			propagatedPermission.EncryptKey = make([]byte, 32)
			propagatedPermission.EncryptKey = permission.EncryptKey
			utils.ApplyPassphrase(masterKey+permission.CreatedBy, propagatedPermission.EncryptKey)
			utils.ApplyPassphrase(masterKey+propagatedPermission.Grantee, propagatedPermission.EncryptKey)
			_, err := addPermissionToObjectInTransaction(tx, childObject, &propagatedPermission, propagateToChildren, masterKey)
			if err != nil {
				return dbPermission, err
			}
		}
		// Additional pages
		for pageNumber := 2; pageNumber < children.PageCount; pageNumber++ {
			pagedChildren, err := getChildObjectsInTransaction(tx, "", pageNumber, MaxPageSize, object)
			if err != nil {
				return dbPermission, err
			}
			for _, childObject := range pagedChildren.Objects {
				propagatedPermission := models.ODObjectPermission{}
				propagatedPermission.CreatedBy = permission.CreatedBy
				// - Same Grantee
				propagatedPermission.Grantee = permission.Grantee
				// - Propogated permissions are not explicit
				propagatedPermission.ExplicitShare = false
				// - Same permissions
				propagatedPermission.AllowCreate = permission.AllowCreate
				propagatedPermission.AllowRead = permission.AllowRead
				propagatedPermission.AllowUpdate = permission.AllowUpdate
				propagatedPermission.AllowDelete = permission.AllowDelete
				propagatedPermission.AllowShare = permission.AllowShare
				// - Encryption
				propagatedPermission.EncryptKey = make([]byte, 32)
				propagatedPermission.EncryptKey = permission.EncryptKey
				utils.ApplyPassphrase(masterKey+permission.CreatedBy, propagatedPermission.EncryptKey)
				utils.ApplyPassphrase(masterKey+propagatedPermission.Grantee, propagatedPermission.EncryptKey)
				_, err := addPermissionToObjectInTransaction(tx, childObject, &propagatedPermission, propagateToChildren, masterKey)
				if err != nil {
					return dbPermission, err
				}
			}
		}
	}

	return dbPermission, nil
}

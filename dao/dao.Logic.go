package dao

import (
	"bytes"

	"github.com/jmoiron/sqlx"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// IsParentIDADescendent accepts an object identifier and a parent that would
// be assigned, and walks the tree from the target parent to the root (nil)
// looking to see if it references the same object.
func (dao *DataAccessLayer) IsParentIDADescendent(id []byte, parentID []byte) (bool, error) {
	defer util.Time("IsParentIDADescendent")()
	tx := dao.MetadataDB.MustBegin()
	result, err := isParentIDADescendentInTransaction(tx, id, parentID)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return result, err
}

func isParentIDADescendentInTransaction(tx *sqlx.Tx, id []byte, parentID []byte) (bool, error) {
	loadPermissions := false
	loadProperties := false
	if parentID == nil {
		return false, nil
	}
	var targetObject models.ODObject
	targetObject.ID = parentID
	dbObject, err := getObjectInTransaction(tx, targetObject, loadPermissions, loadProperties)
	if err != nil {
		return true, err
	}
	if bytes.Compare(dbObject.ParentID, id) == 0 {
		// circular found
		return true, nil
	}
	return isParentIDADescendentInTransaction(tx, id, dbObject.ParentID)
}

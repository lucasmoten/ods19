package dao

import (
	"bytes"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// IsParentIDADescendent accepts an object identifier and a parent that would
// be assigned, and walks the tree from the target parent to the root (nil)
// looking to see if it references the same object.
func IsParentIDADescendent(db *sqlx.DB, id []byte, parentID []byte) (bool, error) {
	if parentID == nil {
		return false, nil
	}
	var targetObject models.ODObject
	targetObject.ID = parentID
	dbObject, err := dao.GetObject(db, &targetObject, false)
	if err != nil {
		return true, err
	}
	if bytes.Compare(dbObject.ParentID, id) == 0 {
		// circular found
		return true, nil
	}
	return IsParentIDADescendent(db, id, dbObject.ParentID)
}

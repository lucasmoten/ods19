package dao

import (
	"bytes"

	"decipher.com/oduploader/metadata/models"
)

// IsParentIDADescendent accepts an object identifier and a parent that would
// be assigned, and walks the tree from the target parent to the root (nil)
// looking to see if it references the same object.
func (dao *DataAccessLayer) IsParentIDADescendent(id []byte, parentID []byte) (bool, error) {
	if parentID == nil {
		return false, nil
	}
	var targetObject models.ODObject
	targetObject.ID = parentID
	dbObject, err := dao.GetObject(&targetObject, false)
	if err != nil {
		return true, err
	}
	if bytes.Compare(dbObject.ParentID, id) == 0 {
		// circular found
		return true, nil
	}
	return dao.IsParentIDADescendent(id, dbObject.ParentID)
}

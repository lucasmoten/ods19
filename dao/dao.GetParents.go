package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetParents iteratively queries up the chain of parents until root is reached, and returns
// a slice of the child's parents. If a root-level object is passed, an empty slice is returned.
func (dao *DataAccessLayer) GetParents(child models.ODObject) ([]models.ODObject, error) {

	parents := make([]models.ODObject, 0)

	if child.ParentID == nil || len(child.ParentID) == 0 {
		return parents, nil
	}

	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return nil, err
	}

	parents, err = getParentsInTransaction(tx, child)

	if err != nil {
		dao.GetLogger().Error("Error in GetParents", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}

	return parents, nil
}

func getParentsInTransaction(tx *sqlx.Tx, child models.ODObject) ([]models.ODObject, error) {
	parents := make([]models.ODObject, 0)

	var queryObj models.ODObject
	queryObj.ID = child.ParentID
	for {

		parent, err := getObjectInTransaction(tx, queryObj, false)
		if err != nil {
			return nil, err
		}

		// Prepend our parent to the slice of parents.
		parents = append([]models.ODObject{parent}, parents...)

		if parent.ParentID == nil || len(parent.ParentID) == 0 {
			// This parent has no parents, so we've arrived at the root.
			break
		}

		queryObj.ID = parent.ParentID
	}

	return parents, nil
}

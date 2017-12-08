package dao

import (
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetParents iteratively queries up the chain of parents until root is reached, and returns
// a slice of the child's parents. If a root-level object is passed, an empty slice is returned.
// The slice of parents is sorted with the root-level parent first, and the object's immediate
// parent last.
func (dao *DataAccessLayer) GetParents(child models.ODObject) ([]models.ODObject, error) {
	defer util.Time("GetParents")()

	var parents []models.ODObject

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
	var parents []models.ODObject

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

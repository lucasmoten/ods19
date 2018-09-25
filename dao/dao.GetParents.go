package dao

import (
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
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
	dao.GetLogger().Debug("dao starting txn for GetParents")
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return nil, err
	}
	dao.GetLogger().Debug("dao passing  txn into getParentsInTransaction")
	parents, err = getParentsInTransaction(tx, child)
	dao.GetLogger().Debug("dao returned txn from getParentsInTransaction")
	if err != nil {
		dao.GetLogger().Debug("dao rolling back txn for GetParents")
		dao.GetLogger().Error("Error in GetParents", zap.Error(err))
		tx.Rollback()
	} else {
		dao.GetLogger().Debug("dao committing txn for GetParents")
		tx.Commit()
	}
	dao.GetLogger().Debug("dao finished txn for GetParents")

	return parents, nil
}

func getParentsInTransaction(tx *sqlx.Tx, child models.ODObject) ([]models.ODObject, error) {
	loadPermissions := true // auth checks in getobject depend on this for determiniing redaction of parents in breadcrumbs
	loadProperties := false
	var parents []models.ODObject
	var queryObj models.ODObject
	queryObj.ID = child.ParentID
	for {
		parent, err := getObjectInTransaction(tx, queryObj, loadPermissions, loadProperties)
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

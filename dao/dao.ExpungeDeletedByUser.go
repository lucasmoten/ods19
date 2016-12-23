package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// ExpungeDeletedByUser for a given user, iterate the list of trashed (deleted) object roots and delete them
func (dao *DataAccessLayer) ExpungeDeletedByUser(user models.ODUser, pageSize int) (int64, error) {
	total := int64(0)

	if pageSize <= 0 {
		pageSize = 10000
	}

	pagingRequest := PagingRequest{
		PageNumber: 1,
		PageSize:   pageSize,
	}

	// Deleting trash can be a huge operation.  Operate in transactional chunks so that we can make progress, even if we time out
	// Note that it's always page 1, because we keep getting the trash list
	for {
		tx, err := dao.MetadataDB.Beginx()
		if err != nil {
			dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
			return total, err
		}
		count, err := dao.expungeDeletedByUserInTransaction(tx, user, pagingRequest)
		total += count
		if err != nil {
			dao.GetLogger().Error("Error in ExpungeDeletedByUser", zap.String("err", err.Error()))
			tx.Rollback()
			return total, err
		}
		tx.Commit()
		// If we deleted 0 objects this time, then we are clean
		if count == 0 {
			return total, nil
		}
	}
}

func (dao *DataAccessLayer) expungeDeletedByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (int64, error) {
	total := int64(0)
	response, err := getTrashedObjectsByUserInTransaction(tx, user, pagingRequest)
	updateObjectStatement, err := expungeObjectInTransactionPrepare(tx)
	defer updateObjectStatement.Close()
	if err != nil {
		return 0, err
	}
	for _, r := range response.Objects {
		//Note: this will do a retrieve of the object by ID!
		err := expungeObjectInTransaction(tx, user, r, true, updateObjectStatement)
		if err != nil {
			return total, err
		}
		total++
	}
	return total, err
}

// Get a page of objects - just the ID because expungeObjectInTransaction does not need a full object
func (dao *DataAccessLayer) expungeDeletedByUserInTransactionMore(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {

	var response models.ODObjectResultset
	var err error

	query := `
    select 
        distinct sql_calc_found_rows 
        o.id,
        o.changeToken    
    from object o
        inner join object_type ot on o.typeid = ot.id
        inner join object_permission op on op.objectid = o.id and op.isdeleted = 0 and op.allowread = 1
        inner join objectacm acm on o.id = acm.objectid
    where o.isdeleted = 1 and o.isExpunged = 0 and o.isAncestorDeleted = 0 `
	query += buildFilterRequireObjectsIOrMyGroupsOwn(tx, user)
	query += buildFilterForUserACMShare(user)
	query += buildFilterForUserSnippets(user)
	query += buildFilterSortAndLimit(pagingRequest)
	err = tx.Select(&response.Objects, query)
	dao.GetLogger().Info("expungeDeletedByUserInTransactionMore", zap.Object("user", user), zap.Object("pagingRequest", pagingRequest), zap.Int("rows", len(response.Objects)))
	return response, err
}

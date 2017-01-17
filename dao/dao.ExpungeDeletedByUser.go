package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// ExpungeDeletedByUser for a given user, iterate the list of trashed (deleted) object roots and delete them
func (dao *DataAccessLayer) ExpungeDeletedByUser(user models.ODUser, pageSize int) (models.ODObjectResultset, error) {

	if pageSize <= 0 {
		pageSize = 10000
	}

	pagingRequest := PagingRequest{
		PageNumber: 1,
		PageSize:   pageSize,
	}

	var overallExpunged models.ODObjectResultset
	overallExpunged.PageCount = 1
	overallExpunged.PageNumber = 1

	// Deleting trash can be a huge operation.  Operate in transactional chunks so that we can make progress, even if we time out
	// Note that it's always page 1, because we keep getting the trash list
	for {
		tx, err := dao.MetadataDB.Beginx()
		if err != nil {
			dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
			return overallExpunged, err
		}
		expungedObjects, err := dao.expungeDeletedByUserInTransaction(tx, user, pagingRequest)
		if err != nil {
			dao.GetLogger().Error("Error in ExpungeDeletedByUser", zap.String("err", err.Error()))
			tx.Rollback()
			return overallExpunged, err
		}
		tx.Commit()
		// If we deleted 0 objects this time, then we are clean
		if expungedObjects.PageCount == 0 {
			return overallExpunged, nil
		}
		for _, r := range expungedObjects.Objects {
			overallExpunged.Objects = append(overallExpunged.Objects, r)
		}
		overallExpunged.PageRows = len(overallExpunged.Objects)
		overallExpunged.PageSize = overallExpunged.PageRows
		overallExpunged.TotalRows = overallExpunged.PageRows
	}
}

func (dao *DataAccessLayer) expungeDeletedByUserInTransaction(tx *sqlx.Tx, user models.ODUser, pagingRequest PagingRequest) (models.ODObjectResultset, error) {
	response, err := getTrashedObjectsByUserInTransaction(tx, user, pagingRequest)
	updateObjectStatement, err := expungeObjectInTransactionPrepare(tx)
	var expungedObjects models.ODObjectResultset
	defer updateObjectStatement.Close()
	if err != nil {
		return expungedObjects, err
	}
	for _, r := range response.Objects {
		//Note: this will do a retrieve of the object by ID!
		err := expungeObjectInTransaction(tx, user, r, true, updateObjectStatement)
		if err != nil {
			return expungedObjects, err
		}
		expungedObjects.Objects = append(expungedObjects.Objects, r)
		expungedObjects.PageNumber = GetSanitizedPageNumber(pagingRequest.PageNumber)
		expungedObjects.PageSize = GetSanitizedPageSize(pagingRequest.PageSize)
		expungedObjects.PageRows = len(expungedObjects.Objects)
		expungedObjects.PageCount = GetPageCount(expungedObjects.TotalRows, expungedObjects.PageSize)
		expungedObjects.TotalRows = expungedObjects.PageRows
	}
	return expungedObjects, err
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

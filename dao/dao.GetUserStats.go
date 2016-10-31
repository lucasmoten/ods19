package dao

import (
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetUserStats returns metrics of object counts and file space used for objects and revisions owned by a user
func (dao *DataAccessLayer) GetUserStats(dn string) (models.UserStats, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.UserStats{}, err
	}
	userStats, err := getUserStatsInTransaction(dao.GetLogger(), tx, dn)
	if err != nil {
		dao.GetLogger().Error("Error in GetUserStats", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return userStats, err
}

func getUserStatsInTransaction(logger zap.Logger, tx *sqlx.Tx, dn string) (models.UserStats, error) {
	var objectStorageMetrics []models.UserStatsMetrics
	var userStats models.UserStats
	objectsIOwn := buildFilterRequireObjectsIOwn(models.ODUser{DistinguishedName: dn})
	sqlStatement := `
    select 
        t.name as TypeName,
        (select count(o.id) from object as o where o.typeid=t.id ` + objectsIOwn + `) as Objects, 
        ifnull((select sum(o.contentSize) from object as o where o.typeid=t.id ` + objectsIOwn + `),0) as ObjectsSize,
        (select count(ao.id) from a_object as ao where ao.typeid=t.id ` + asArchive(objectsIOwn) + `) as ObjectsAndRevisions,
        ifnull((select sum(ao.contentSize) from a_object as ao where ao.typeid=t.id ` + asArchive(objectsIOwn) + `),0) as ObjectsAndRevisionsSize
    from 
        object_type as t
    group by t.name
    `
	err := tx.Select(&objectStorageMetrics, sqlStatement)
	if err != nil {
		logger.Error("Unable to execute query", zap.String("sql", sqlStatement), zap.String("err", err.Error()))
		return userStats, err
	}
	userStats.ObjectStorageMetrics = objectStorageMetrics
	for i := range userStats.ObjectStorageMetrics {
		userStats.TotalObjects += userStats.ObjectStorageMetrics[i].Objects
		userStats.TotalObjectsAndRevisions += userStats.ObjectStorageMetrics[i].ObjectsAndRevisions
		userStats.TotalObjectsSize += userStats.ObjectStorageMetrics[i].ObjectsSize
		userStats.TotalObjectsAndRevisionsSize += userStats.ObjectStorageMetrics[i].ObjectsAndRevisionsSize
	}
	return userStats, nil
}

package dao

import (
	"strings"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetUserStats returns metrics of object counts and file space used for objects and revisions owned by a user
func (dao *DataAccessLayer) GetUserStats(dn string) (models.UserStats, error) {
	defer util.Time("GetUserStats")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.Error(err))
		return models.UserStats{}, err
	}
	userStats, err := getUserStatsInTransaction(dao.GetLogger(), tx, dn)
	if err != nil {
		dao.GetLogger().Error("Error in GetUserStats", zap.Error(err))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return userStats, err
}

func getUserStatsInTransaction(logger *zap.Logger, tx *sqlx.Tx, dn string) (models.UserStats, error) {
	var userStats models.UserStats
	var err error
	// Get base objects
	var objectMetrics []models.UserStatsMetrics
	sql := `
		select 
			t.name as TypeName, 
			count(o.id) as Objects, 
			sum(ifnull(o.contentsize,0)) as ObjectsSize
		from
			object_type t 
			inner join object o on t.id = o.typeid ` + buildFilterRequireObjectsIOwn(tx, models.ODUser{DistinguishedName: dn}) + `
		group by 
			t.name`
	err = tx.Select(&objectMetrics, sql)
	if err != nil {
		logger.Error("Unable to execute query", zap.String("sql", sql), zap.Error(err))
		return userStats, err
	}
	userStats.ObjectStorageMetrics = objectMetrics
	// Get archive
	var archiveMetrics []models.UserStatsMetrics
	sql = strings.Replace(sql, " object ", " a_object ", -1)
	err = tx.Select(&archiveMetrics, sql)
	if err != nil {
		logger.Error("Unable to execute query", zap.String("sql", sql), zap.Error(err))
		return userStats, err
	}
	// Merge archive into base objects
	for _, archive := range archiveMetrics {
		archiveTypeFound := false
		for i, metric := range userStats.ObjectStorageMetrics {
			if archive.TypeName == metric.TypeName {
				userStats.ObjectStorageMetrics[i].ObjectsAndRevisions = archive.Objects
				userStats.ObjectStorageMetrics[i].ObjectsAndRevisionsSize = archive.ObjectsSize
				archiveTypeFound = true
			}
		}
		if !archiveTypeFound {
			userStats.ObjectStorageMetrics = append(userStats.ObjectStorageMetrics, models.UserStatsMetrics{TypeName: archive.TypeName, Objects: 0, ObjectsSize: 0, ObjectsAndRevisions: archive.Objects, ObjectsAndRevisionsSize: archive.ObjectsSize})
		}
	}
	// Get overall totals
	for i := range userStats.ObjectStorageMetrics {
		userStats.TotalObjects += userStats.ObjectStorageMetrics[i].Objects
		userStats.TotalObjectsAndRevisions += userStats.ObjectStorageMetrics[i].ObjectsAndRevisions
		userStats.TotalObjectsSize += userStats.ObjectStorageMetrics[i].ObjectsSize
		userStats.TotalObjectsAndRevisionsSize += userStats.ObjectStorageMetrics[i].ObjectsAndRevisionsSize
	}
	return userStats, nil
}

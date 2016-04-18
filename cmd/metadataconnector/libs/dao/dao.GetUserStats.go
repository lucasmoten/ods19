package dao

import (
	"log"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

func (dao *DataAccessLayer) GetUserStats(dn string) (models.UserStats, error) {

	tx := dao.MetadataDB.MustBegin()
	userStats, err := getUserStatsInTransaction(tx, dn)
	if err != nil {
		log.Printf("Error in UserStats: %v\n", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return userStats, err
}

func getUserStatsInTransaction(tx *sqlx.Tx, dn string) (models.UserStats, error) {
	var objectStorageMetrics []models.UserStatsMetrics
	var userStats models.UserStats
	sqlStatement := `
    select 
      t.name as TypeName,
      (select count(o.id) from object as o where o.typeid=t.id and o.ownedby=? ) as Objects,
      ifnull((select sum(o.contentSize) from object as o where o.typeid=t.id and o.ownedby=? ),0) as ObjectsSize,
      (select count(ao.id) from a_object as ao where ao.typeid=t.id and ao.ownedby=? ) as ObjectsAndRevisions,
      ifnull((select sum(ao.contentSize) from a_object as ao where ao.typeid=t.id and ao.ownedby=? ),0) as ObjectsAndRevisionsSize
    from object_type as t
    group by t.name
    `
	err := tx.Select(&objectStorageMetrics, sqlStatement, dn, dn, dn, dn)
	if err != nil {
		log.Printf("Unable to execute query %s:%v", sqlStatement, err)
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

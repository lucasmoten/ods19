package dao

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/metadata/models/acm"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GetUserAOCacheByDistinguishedName looks up the user authorization object cache state using the
// provided distinguished name
func (dao *DataAccessLayer) GetUserAOCacheByDistinguishedName(user models.ODUser) (models.ODUserAOCache, error) {
	defer util.Time("GetUserAOCacheByDistinguishedName")()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODUserAOCache{}, err
	}
	dbUserAOCache, err := getUserAOCacheByDistinguishedNameInTransaction(tx, user)
	if err != nil {
		if err != sql.ErrNoRows {
			dao.GetLogger().Error("Error in GetUserAOCacheByDistinguishedName", zap.String("err", err.Error()))
		} else {
			err = nil
		}
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbUserAOCache, err
}

func getUserAOCacheByDistinguishedNameInTransaction(tx *sqlx.Tx, user models.ODUser) (models.ODUserAOCache, error) {
	var dbUserAOCache models.ODUserAOCache
	stmt := `select uaoc.id, uaoc.userid, uaoc.iscaching, uaoc.cachedate, uaoc.sha256hash
			from useraocache uaoc inner join user u on uaoc.userid = u.id
			where u.distinguishedName = ?`
	err := tx.Get(&dbUserAOCache, stmt, user.DistinguishedName)
	return dbUserAOCache, err
}

// SetUserAOCacheByDistinguishedName ensures a record exists for the user authorization object cache state,
// marks it as being cached, and rebuilds the cache parts for the authorization object from snippets
func (dao *DataAccessLayer) SetUserAOCacheByDistinguishedName(useraocache *models.ODUserAOCache, user models.ODUser) error {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	// Uninitialized passed in. New it up
	if useraocache == nil {
		newuseraocache := models.ODUserAOCache{}
		newuseraocache.UserID = user.ID
		newuseraocache.CacheDate.Time = time.Now()
		newuseraocache.CacheDate.Valid = true
		newuseraocache.IsCaching = true
		newuseraocache.SHA256Hash = "--uninitialized-user-ao-cache!!!--"
		useraocache = &newuseraocache
	}
	if len(useraocache.UserID) == 0 {
		useraocache.UserID = user.ID
	}
	// Check if first insert or otherwise updates
	if useraocache.ID == 0 {
		err = insertUserAOCache(dao, tx, useraocache)
	} else {
		err = updateUserAOCache(dao, tx, useraocache)
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	// Add any missing keys and values
	if err = createKeysAndValuesFromSnippets(dao, user.Snippets.Snippets); err != nil {
		return err
	}
	// delete existing
	if err := execStatementWithDeadlockRetry(dao, "deleteUserAOCacheParts", "delete from useraocachepart where userid = unhex('"+hex.EncodeToString(user.ID)+"')"); err != nil {
		return err
	}
	// Add the definition parts to the user
	if err = insertUserAOCacheParts(dao, user); err != nil {
		return err
	}
	return nil
}

// createKeysAndValuesForSnippets takes a more tactical approach to adding rows to the database to
// support referential integrity by iterating the definition, and building up minimal SQL statements
// before hitting the database, as opposed to traditional round trip select/insert calls for each
// unique key or value.
func createKeysAndValuesFromSnippets(dao *DataAccessLayer, snippetFields []acm.RawSnippetFields) error {
	// PREP
	// tables to insert rows
	keySQL := `insert into acmkey2 (name) values `
	valueSQL := `insert into acmvalue2 (name) values `
	// value pairings
	kf, vf := false, false
	for _, f := range snippetFields {
		if len(f.FieldName) > 0 {
			if kf {
				keySQL += `,`
			}
			kf = true
			keySQL += `('` + MySQLSafeString2(f.FieldName) + `')`
		}
		for _, v := range f.Values {
			if vf {
				valueSQL += `,`
			}
			vf = true
			valueSQL += `('` + MySQLSafeString2(v) + `')`
		}
	}
	// closeout with duplicate handler (keep as same)
	keySQL += ` on duplicate key update name = name`
	valueSQL += ` on duplicate key update name = name`
	// run them
	if err := execStatementWithDeadlockRetry(dao, "createKeysFromSnippets", keySQL); err != nil {
		return err
	}
	if err := execStatementWithDeadlockRetry(dao, "createValuesFromSnippets", valueSQL); err != nil {
		return err
	}
	return nil
}

func insertUserAOCacheParts(dao *DataAccessLayer, user models.ODUser) error {
	snippetFields := user.Snippets.Snippets
	fullsql := `insert into useraocachepart (userid, isallowed, userkeyid, uservalueid) `
	for i, f := range snippetFields {
		sql := ``
		if i > 0 {
			sql += ` union `
		}
		sql += `select unhex('` + hex.EncodeToString(user.ID) + `')`
		if f.Treatment == "allowed" {
			sql += fmt.Sprintf(",1,ak%d.id,", i)
		} else {
			sql += fmt.Sprintf(",0,ak%d.id,", i)
		}
		if len(f.Values) == 0 {
			sql += fmt.Sprintf("null from acmkey2 ak%d where ak%d.name = '%s'", i, i, MySQLSafeString2(f.FieldName))
		} else {
			sql += fmt.Sprintf("av%d.id from acmkey2 ak%d left outer join acmvalue2 av%d on 1=1 where ak%d.name = '%s' and av%d.name in (", i, i, i, i, MySQLSafeString2(f.FieldName), i)
			for x, v := range f.Values {
				if x > 0 {
					sql += `,`
				}
				sql += fmt.Sprintf("'%s'", MySQLSafeString2(v))
			}
			sql += `)`
		}
		fullsql += sql
	}
	return execStatementWithDeadlockRetry(dao, "insertUserAOCacheParts", fullsql)
}

func execStatementWithDeadlockRetry(dao *DataAccessLayer, funcLbl string, sql string) error {
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	deadlockMessage := "Deadlock"
	logger := dao.GetLogger()

	// Initial transaction
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		return fmt.Errorf("%s could not begin transaction, %s", funcLbl, err.Error())
	}
	// deadlock looper
	err = fmt.Errorf(deadlockMessage)
	for retryCounter > 0 && err != nil && strings.Contains(err.Error(), deadlockMessage) {
		if strings.Contains(err.Error(), deadlockMessage) && retryCounter != dao.DeadlockRetryCounter {
			logger.Info("deadlock, restarting transaction", zap.String("funcLbl", funcLbl), zap.Int64("retryCounter", retryCounter))
		}
		tx.Rollback()
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			return fmt.Errorf("%s could not begin transaction, %s", funcLbl, err.Error())
		}
		retryCounter--
		stmt, err := tx.Preparex(sql)
		if err != nil {
			return fmt.Errorf("%s error preparing key statement, %s", funcLbl, err.Error())
		}
		err = nil
		_, err = stmt.Exec()
	}
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("%s error executing statement, %s", funcLbl, err.Error())
	}
	tx.Commit()
	return nil
}

func insertUserAOCache(dao *DataAccessLayer, tx *sqlx.Tx, useraocache *models.ODUserAOCache) error {
	useraocache.CacheDate.Time = time.Now()
	useraocache.CacheDate.Valid = true
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	deadlockMessage := "Deadlock"
	logger := dao.GetLogger()

	stmt, err := tx.Preparex(`insert useraocache set userid = ?, iscaching = ?, cachedate = ?, sha256hash = ?`)
	if err != nil {
		return fmt.Errorf("insertUserAOCache error preparing add statement, %s", err.Error())
	}
	result, err := stmt.Exec(useraocache.UserID, useraocache.IsCaching, useraocache.CacheDate, useraocache.SHA256Hash)
	for retryCounter > 0 && err != nil && strings.Contains(err.Error(), deadlockMessage) {
		if strings.Contains(err.Error(), deadlockMessage) {
			logger.Info("deadlock in insertUserAOCache, restarting transaction", zap.Int64("retryCounter", retryCounter))
		}
		tx.Rollback()
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction", zap.String("err", err.Error()))
			return err
		}
		stmt, err = tx.Preparex(`insert useraocache set userid = ?, iscaching = ?, cachedate = ?, sha256hash = ?`)
		if err != nil {
			return fmt.Errorf("insertUserAOCache error preparing add statement after deadlock, %s", err.Error())
		}
		// Retry
		retryCounter--
		result, err = stmt.Exec(useraocache.UserID, useraocache.IsCaching, useraocache.CacheDate, useraocache.SHA256Hash)
	}
	if err != nil {
		return fmt.Errorf("insertUserAOCache error executing add statement, %s", err.Error())
	}
	useraocache.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("insertUserAOCache error getting last inserted id, %s", err.Error())
	}
	return nil
}

func updateUserAOCache(dao *DataAccessLayer, tx *sqlx.Tx, useraocache *models.ODUserAOCache) error {
	useraocache.CacheDate.Time = time.Now()
	useraocache.CacheDate.Valid = true
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	deadlockMessage := "Deadlock"
	logger := dao.GetLogger()

	stmt, err := tx.Preparex(`update useraocache set iscaching = ?, cachedate = ?, sha256hash = ? where userid = ?`)
	if err != nil {
		return fmt.Errorf("updateUserAOCache error preparing update statement, %s", err.Error())
	}
	result, err := stmt.Exec(useraocache.IsCaching, useraocache.CacheDate, useraocache.SHA256Hash, useraocache.UserID)
	for retryCounter > 0 && err != nil && strings.Contains(err.Error(), deadlockMessage) {
		if strings.Contains(err.Error(), deadlockMessage) {
			logger.Info("deadlock in updateUserAOCache, restarting transaction", zap.Int64("retryCounter", retryCounter))
		}
		tx.Rollback()
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction", zap.String("err", err.Error()))
			return err
		}
		stmt, err = tx.Preparex(`update useraocache set iscaching = ?, cachedate = ?, sha256hash = ? where userid = ?`)
		if err != nil {
			return fmt.Errorf("updateUserAOCache error preparing update statement after deadlock, %s", err.Error())
		}
		// Retry
		retryCounter--
		result, err = stmt.Exec(useraocache.IsCaching, useraocache.CacheDate, useraocache.SHA256Hash, useraocache.UserID)
	}
	if err != nil {
		return fmt.Errorf("updateUserAOCache error executing update statement, %s", err.Error())
	}
	ra, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updateUserAOCache error determining rows affected, %s", err.Error())
	}
	if ra == 0 {
		return fmt.Errorf("udpateUserAOCache did not affect any rows")
	}
	return nil
}

func deleteUserAOCacheParts(tx *sqlx.Tx, user models.ODUser) error {
	sql := `delete from useraocachepart where userid = ?`
	stmt, err := tx.Preparex(sql)
	if err != nil {
		return fmt.Errorf("deleteUserAOCachePart error preparing delete statement, %s", err.Error())
	}
	result, err := stmt.Exec(user.ID)
	if err != nil {
		return fmt.Errorf("deleteuseraocachepart error executing delete statement, %s", err.Error())
	}
	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleteuseraocachepart error determining rows affected, %s", err.Error())
	}
	return nil
}

func insertUserAOCachePart(dao *DataAccessLayer, tx *sqlx.Tx, user models.ODUser, isallowed bool, acmkey models.ODAcmKey2, acmvalue *models.ODAcmValue2) error {
	deadlockRetryCounter := dao.DeadlockRetryCounter
	deadlockRetryDelay := dao.DeadlockRetryDelay
	deadlockMessage := "Deadlock"
	logger := dao.GetLogger()
	var result sql.Result
	var err error
	stmt, err := tx.Preparex(`insert useraocachepart set userid = ?, isallowed = ?, userkeyid = ?, uservalueid = ?`)
	if err != nil {
		return fmt.Errorf("insertUserAOCachePart error preparing add statement, %s", err.Error())
	}
	if acmvalue != nil {
		result, err = stmt.Exec(user.ID, isallowed, acmkey.ID, acmvalue.ID)
	} else {
		result, err = stmt.Exec(user.ID, isallowed, acmkey.ID, nil)
	}
	for deadlockRetryCounter > 0 && err != nil && strings.Contains(err.Error(), deadlockMessage) {
		if strings.Contains(err.Error(), deadlockMessage) {
			logger.Info("deadlock in insertUserAOCachePart, restarting transaction", zap.Int64("deadlockRetryCounter", deadlockRetryCounter))
		}
		tx.Rollback()
		time.Sleep(time.Duration(deadlockRetryDelay) * time.Millisecond)
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction in useraocachepart", zap.String("err", err.Error()))
			return err
		}
		stmt, err = tx.Preparex(`insert useraocachepart set userid = ?, isallowed = ?, userkeyid = ?, uservalueid = ?`)
		if err != nil {
			return fmt.Errorf("insertUserAOCachePart error preparing add statement after deadlock, %s", err.Error())
		}
		// Retry
		deadlockRetryCounter--
		if acmvalue != nil {
			result, err = stmt.Exec(user.ID, isallowed, acmkey.ID, acmvalue.ID)
		} else {
			result, err = stmt.Exec(user.ID, isallowed, acmkey.ID, nil)
		}
	}
	if err != nil {
		return fmt.Errorf("insertUserAOCachePart error executing add statement, %s", err.Error())
	}
	ra, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("insertUserAOCachePart error determining rows affected, %s", err.Error())
	}
	if ra == 0 {
		return fmt.Errorf("insertUserAOCachePart did not affect any rows, %s", err.Error())
	}
	return nil
}

// RebuildUserACMCache examines the user authorization object cache parts and compares to acms to determine which
// acms the user is elligible to see, and then forms a static link for use for fast filtering in search/list calls
func (dao *DataAccessLayer) RebuildUserACMCache(useraocache *models.ODUserAOCache, user models.ODUser, done chan bool, mode string) error {
	defer func() { done <- true }()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	// 1. Determine the ids a user should have
	var acmids []int64
	if acmids, err = getACMIDsValidForUser(tx, user, mode); err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache had error getting list of matching ids", zap.Error(err))
		return err
	}
	acmidlist := sqlIntSeq(acmids)
	// 2. Delete those which the user currently has associated that should not be anymore
	if err = deleteInvalidUserACMs(tx, user, acmidlist); err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache error deleting existing user acms", zap.Error(err))
		return err
	}
	// 3. Insert those that the user should have but currently dont
	if err = insertUserACMList(tx, user, acmidlist); err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache error inserting useracms", zap.String("acmidlist", acmidlist), zap.Error(err))
		return err
	}
	tx.Commit()
	// 4. Done caching
	tx, err = dao.MetadataDB.Beginx()
	if err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache Could not begin transaction for marking cache complete", zap.String("err", err.Error()))
		return err
	}
	useraocache.IsCaching = false
	if err = updateUserAOCache(dao, tx, useraocache); err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache error marking user cache as done", zap.Error(err))
		return err
	}
	tx.Commit()
	return nil
}

// AssociateUsersToNewACM examines the user authorization cache parts relative to the definition of the
// acm identified by the passed in identifier and its associated parts to determine which users should
// have the acm association, and then links them
func (dao *DataAccessLayer) AssociateUsersToNewACM(object models.ODObject, done chan bool) error {
	defer func() { done <- true }()
	if err := associateUsersToNewACM(dao, object, 0); err != nil {
		dao.GetLogger().Error("associateUsersToNewACM encountered error", zap.Error(err))
		return err
	}
	return nil
}
func associateUsersToNewACM(dao *DataAccessLayer, object models.ODObject, retryCount int) error {
	maxRetry := 5
	// 0. Start a new transaction on each retry to take into account new snapshots from background
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		return err
	}

	// 1. Get elligible users
	var users []models.ODUser
	if users, err = getUsersValidForACMByID(tx, object.ACMID); err != nil {
		tx.Rollback()
		return err
	}
	// 2. Must be at least one user, try again
	if len(users) == 0 {
		tx.Rollback()
		if retryCount > maxRetry {
			return fmt.Errorf("no matching users after %d attempts", maxRetry)
		}
		time.Sleep(time.Millisecond * 100)
		return associateUsersToNewACM(dao, object, retryCount+1)
	}
	tx.Commit()

	// 3. With each user, add association to the acm
	for _, user := range users {
		txUserACM, err := dao.MetadataDB.Beginx()
		if err != nil {
			return err
		}
		if err := insertUserACM(txUserACM, user, object.ACMID); err != nil {
			txUserACM.Rollback()
			return err
		}
		txUserACM.Commit()
	}
	return nil
}

type useraocachepartsnippet struct {
	Key       string `db:"keyname"`
	IsAllowed bool   `db:"isallowed"`
	Value     string `db:"valuename"`
}

func getUserAOCachePartSnippets(tx *sqlx.Tx, user models.ODUser) ([]useraocachepartsnippet, error) {
	var cachesnippets []useraocachepartsnippet
	sql := `select ak.name keyname, uaocp.isallowed isallowed, IFNULL(av.name,'') valuename from useraocachepart uaocp inner join acmkey2 ak on uaocp.userkeyid = ak.id left outer join acmvalue2 av on uaocp.uservalueid = av.id where uaocp.userid = ? order by 1,2,3`
	stmt, err := tx.Preparex(sql)
	if err != nil {
		return cachesnippets, fmt.Errorf("getUserAOCachePartSnippets error preparing select statement, %s", err.Error())
	}
	if err = stmt.Select(&cachesnippets, user.ID); err != nil {
		return cachesnippets, fmt.Errorf("getUserAOCachePartSnippets error executing select statement, %s", err.Error())
	}
	return cachesnippets, nil
}

func getACMIDsValidForUser(tx *sqlx.Tx, user models.ODUser, mode string) ([]int64, error) {
	var acmids []int64
	snippets, err := getUserSnippets(tx, user)
	if err != nil {
		return acmids, err
	}
	acmids, err = getACMIDsValidForUserBySnippets(tx, snippets, user, mode)
	return acmids, err
}

func getUserSnippets(tx *sqlx.Tx, user models.ODUser) (*acm.ODriveRawSnippetFields, error) {
	if user.Snippets != nil {
		return user.Snippets, nil
	}
	var cachesnippets []useraocachepartsnippet
	var err error
	if cachesnippets, err = getUserAOCachePartSnippets(tx, user); err != nil {
		return nil, err
	}
	snippets := convertCacheSnippetsToODriveRawSnippetFields(cachesnippets)
	return snippets, nil
}

func convertCacheSnippetsToODriveRawSnippetFields(cachesnippets []useraocachepartsnippet) *acm.ODriveRawSnippetFields {
	var snippets acm.ODriveRawSnippetFields
	key := ""
	for _, cachesnippet := range cachesnippets {
		if key != cachesnippet.Key {
			rawSnippetFields := acm.RawSnippetFields{FieldName: cachesnippet.Key}
			if cachesnippet.IsAllowed {
				rawSnippetFields.Treatment = "allowed"
			} else {
				rawSnippetFields.Treatment = "disallow"
			}
			snippets.Snippets = append(snippets.Snippets, rawSnippetFields)
			key = cachesnippet.Key
		}
		idx := len(snippets.Snippets) - 1
		if len(cachesnippet.Value) > 0 {
			snippets.Snippets[idx].Values = append(snippets.Snippets[idx].Values, cachesnippet.Value)
		}
	}
	return &snippets
}

func getACMIDsValidForUserBySnippets(tx *sqlx.Tx, snippets *acm.ODriveRawSnippetFields, user models.ODUser, mode string) ([]int64, error) {
	var acmids []int64
	var sql string
	sql += "select distinct id from acm2 where 1=1 "
	// Now iterate all the fields building up the where clause portion
	for _, rawFields := range snippets.Snippets {
		switch rawFields.Treatment {
		case "disallow":
			sql += " and id not in ("
			// where it does have the field
			sql += "select acmid from acmpart2 inner join acmkey2 on acmpart2.acmkeyid = acmkey2.id inner join acmvalue2 on acmpart2.acmvalueid = acmvalue2.id "
			sql += "where acmkey2.name = '" + MySQLSafeString2(rawFields.FieldName) + "' "
			sql += "and acmvalue2.name in (''"
			for _, value := range rawFields.Values {
				sql += ",'" + MySQLSafeString2(value) + "'"
			}
			sql += ")) "
		case "allowed":
			sql += " and id in ("
			// where it doesn't have the field
			sql += "select id from acm2 where id not in (select acm2.id from acm2 inner join acmpart2 on acmpart2.acmid = acm2.id inner join acmkey2 on acmpart2.acmkeyid = acmkey2.id and acmkey2.name like '" + MySQLSafeString(rawFields.FieldName) + "')"
			// where it does have the field
			sql += " union "
			sql += "select acmid from acmpart2 inner join acmkey2 on acmpart2.acmkeyid = acmkey2.id inner join acmvalue2 on acmpart2.acmvalueid = acmvalue2.id "
			sql += "where acmkey2.name = '" + MySQLSafeString2(rawFields.FieldName) + "' "
			sql += "and (acmvalue2.name = '' "
			for _, value := range rawFields.Values {
				sql += " or acmvalue2.name = '" + MySQLSafeString2(value) + "'"
			}
			sql += "))"
		default:
			return acmids, fmt.Errorf("unhandled treatment type from snippets %s", rawFields.Treatment)
		}
	}
	if mode == "userroot" {
		ownedbyid := getACMValueFor(tx, models.AACFlatten(user.DistinguishedName))
		sql += fmt.Sprintf(" and id in (select distinct acmid from object where parentid is null and ownedbyid = %d)", ownedbyid)
	}
	if err := tx.Select(&acmids, sql); err != nil {
		return acmids, err
	}
	return acmids, nil
}

func sqlIntSeq(ns []int64) string {
	if len(ns) == 0 {
		return ""
	}

	// Appr. 3 chars per num plus the comma.
	estimate := len(ns) * 4
	b := make([]byte, 0, estimate)
	// Or simply
	//   b := []byte{}
	for _, n := range ns {
		b = strconv.AppendInt(b, int64(n), 10)
		b = append(b, ',')
	}
	b = b[:len(b)-1]
	return string(b)
}

func deleteInvalidUserACMs(tx *sqlx.Tx, user models.ODUser, acmidlist string) error {
	sql := `delete from useracm where userid = unhex('` + hex.EncodeToString(user.ID) + `')`
	if len(acmidlist) > 0 {
		// no valid acmids for user, delete them all
		sql += ` and acmid not in (` + acmidlist + `)`
	}
	stmt, err := tx.Preparex(sql)
	if err != nil {
		return fmt.Errorf("deleteUserACMs error preparing delete statement, %s", err.Error())
	}
	result, err := stmt.Exec()
	if err != nil {
		return fmt.Errorf("deleteUserACMs error executing delete statement, %s", err.Error())
	}
	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleteUserACMs error determining rows affected, %s", err.Error())
	}
	return nil
}

func insertUserACMList(tx *sqlx.Tx, user models.ODUser, acmidlist string) error {
	if len(acmidlist) == 0 {
		// no acmids for user, nothing to do
		return nil
	}
	sql := `insert into useracm (userid,acmid) select unhex('`
	sql += hex.EncodeToString(user.ID) + `'), id `
	sql += `from acm2 where id not in (select acmid from useracm where userid = unhex('`
	sql += hex.EncodeToString(user.ID) + `')) and id in (` + acmidlist + `)`
	stmt, err := tx.Preparex(sql)
	if err != nil {
		return fmt.Errorf("insertUserACMList error preparing insert statement, %s", err.Error())
	}
	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("insertUserACMList error executing insert statement, %s", err.Error())
	}
	return nil
}

func insertUserACM(tx *sqlx.Tx, user models.ODUser, acmID int64) error {
	return insertUserACMList(tx, user, fmt.Sprintf("%v", acmID))
}

type acmkv struct {
	Key   int64 `db:"acmkey"`
	Value int64 `db:"acmvalue"`
}

func insertAssociationOfACMToModifiedByIfValid(dao *DataAccessLayer, object models.ODObject) error {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		return err
	}
	// Get user id for the modifier of the object
	user := models.ODUser{DistinguishedName: object.ModifiedBy}
	user, err = getUserByDistinguishedNameInTransaction(tx, user)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Join to ACM. User permission to this acm was previously validated
	if err = insertUserACM(tx, user, object.ACMID); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func getUsersValidForACMByID(tx *sqlx.Tx, acmID int64) ([]models.ODUser, error) {
	var users []models.ODUser
	var stmt *sqlx.Stmt
	var err error

	sql, err := buildUsersValidForACMByIDSQL(tx, acmID)
	if err != nil {
		return users, fmt.Errorf("getUsersValidForACMByID error building select statement, %s", err.Error())
	}
	if stmt, err = tx.Preparex(sql); err != nil {
		return users, fmt.Errorf("getUsersValidForACMByID error preparing select statement for users, %s", err.Error())
	}
	if err = stmt.Select(&users); err != nil {
		return users, fmt.Errorf("getUsersValidForACMByID error executing select statement for users, %s", err.Error())
	}

	return users, nil
}

func buildUsersValidForACMByIDSQL(tx *sqlx.Tx, acmID int64) (string, error) {
	var acmkvs []acmkv
	var err error
	var stmt *sqlx.Stmt
	acmsql := `select ak.id acmkey, av.id acmvalue from acm2 a inner join acmpart2 ap on a.id = ap.acmid inner join acmkey2 ak on ap.acmkeyid = ak.id inner join acmvalue2 av on ap.acmvalueid = av.id where a.id = ?`
	if stmt, err = tx.Preparex(acmsql); err != nil {
		return "", fmt.Errorf("getUsersValidForACMByID error preparing select statement for acm, %s", err.Error())
	}
	if err = stmt.Select(&acmkvs, acmID); err != nil {
		return "", fmt.Errorf("getUsersValidForACMByID error executing select statement for acm, %s", err.Error())
	}

	keywhitelist := []int64{}
	keyblacklist := []int64{}
	sql := "select distinct u.id id from user u inner join useraocachepart uaocp on u.id = uaocp.userid where 1=1"
	var disallowunionpart string
	var allowselectpart string
	for _, acmkv := range acmkvs {
		// open key
		sql += " and u.id in ("
		// allowed
		allowedkeyprocessed := false
		for _, i := range keywhitelist {
			if i == acmkv.Key {
				allowedkeyprocessed = true
				break
			}
		}
		if !allowedkeyprocessed {
			allowselectpart = fmt.Sprintf("select userid from useraocachepart where isallowed = 1 and userkeyid = %d and uservalueid in (-1", acmkv.Key)
			for _, acmkvvalues := range acmkvs {
				if acmkvvalues.Key == acmkv.Key {
					allowselectpart += fmt.Sprintf(",%d", acmkvvalues.Value)
				}
			}
			allowselectpart += ")"
			keywhitelist = append(keywhitelist, acmkv.Key)
		}
		sql += allowselectpart
		// disallow
		disallowkeyprocessed := false
		for _, i := range keyblacklist {
			if i == acmkv.Key {
				disallowkeyprocessed = true
				break
			}
		}
		if !disallowkeyprocessed {
			disallowunionpart = " union "
			disallowunionpart += fmt.Sprintf("select userid from useraocachepart where isallowed = 0 and userkeyid = %d and (uservalueid is null or (1=1", acmkv.Key)
			for _, acmkvvalues := range acmkvs {
				if acmkvvalues.Key == acmkv.Key {
					disallowunionpart += fmt.Sprintf(" and uservalueid <> %d", acmkvvalues.Value)
				}
			}
			disallowunionpart += "))"
			keyblacklist = append(keyblacklist, acmkv.Key)
		}
		sql += disallowunionpart
		// close key
		sql += ")"
	}
	return sql, nil
}

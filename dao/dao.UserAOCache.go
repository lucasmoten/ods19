package dao

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// GetUserAOCacheByDistinguishedName looks up the user authorization object cache state using the
// provided distinguished name
func (dao *DataAccessLayer) GetUserAOCacheByDistinguishedName(user models.ODUser) (models.ODUserAOCache, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return models.ODUserAOCache{}, err
	}
	dbUserAOCache, err := getUserAOCacheByDistinguishedNameInTransaction(tx, user)
	if err != nil {
		if err != sql.ErrNoRows {
			dao.GetLogger().Error("Error in GetUserAOCacheByDistinguishedName", zap.String("err", err.Error()))
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
		newuseraocache.IsCaching = true
		newuseraocache.SHA256Hash = ""
		useraocache = &newuseraocache
	}
	// Check if first insert or otherwise updates
	if useraocache.ID == 0 {
		err = insertUserAOCache(tx, useraocache)
	} else {
		err = updateUserAOCache(tx, useraocache)
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	// New transaction..
	tx, err = dao.MetadataDB.Beginx()
	// Clear user ao cache parts
	err = deleteUserAOCacheParts(tx, user)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Build user ao cache parts
	for _, snippet := range user.Snippets.Snippets {
		acmkey, err := getAcmKey2ByNameInTransaction(tx, snippet.FieldName, true)
		if err != nil {
			tx.Rollback()
			return err
		}
		isallowed := (snippet.Treatment == "allowed")
		if len(snippet.Values) > 0 {
			for _, value := range snippet.Values {
				acmvalue, err := getAcmValue2ByNameInTransaction(tx, value, true)
				if err != nil {
					tx.Rollback()
					return err
				}
				err = insertUserAOCachePart(tx, user, isallowed, acmkey, &acmvalue)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		} else {
			// Definition can include no values which means..
			//  allowed -- no values are allowed
			//  disallow -- no values are prevented
			err = insertUserAOCachePart(tx, user, isallowed, acmkey, nil)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	tx.Commit()

	return nil
}

func insertUserAOCache(tx *sqlx.Tx, useraocache *models.ODUserAOCache) error {
	stmt, err := tx.Preparex(`insert useraocache set userid = ?, iscaching = ?, cachedate = ?, sha256hash = ?`)
	if err != nil {
		return fmt.Errorf("insertUserAOCache error preparing add statement, %s", err.Error())
	}
	result, err := stmt.Exec(useraocache.UserID, useraocache.IsCaching, useraocache.CacheDate, useraocache.SHA256Hash)
	if err != nil {
		return fmt.Errorf("insertUserAOCache error executing add statement, %s", err.Error())
	}
	useraocache.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("insertUserAOCache error getting last inserted id, %s", err.Error())
	}
	return nil
}

func updateUserAOCache(tx *sqlx.Tx, useraocache *models.ODUserAOCache) error {
	stmt, err := tx.Preparex(`update useraocache set iscaching = ?, cachedate = ?, sha256hash = ? where userid = ?`)
	if err != nil {
		return fmt.Errorf("updateUserAOCache error preparing update statement, %s", err.Error())
	}
	result, err := stmt.Exec(useraocache.IsCaching, useraocache.CacheDate, useraocache.SHA256Hash, useraocache.UserID)
	if err != nil {
		return fmt.Errorf("updateUserAOCache error executing update statement, %s", err.Error())
	}
	ra, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updateUserAOCache error determining rows affected, %s", err.Error())
	}
	if ra == 0 {
		return fmt.Errorf("udpateUserAOCache did not affect any rows, %s", err.Error())
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

func insertUserAOCachePart(tx *sqlx.Tx, user models.ODUser, isallowed bool, acmkey models.ODAcmKey2, acmvalue *models.ODAcmValue2) error {
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
func (dao *DataAccessLayer) RebuildUserACMCache(useraocache *models.ODUserAOCache, user models.ODUser, done chan bool) error {
	defer func() { done <- true }()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	// Rebuilding based upon user.snippets for now.
	// 1. Delete existing User ACM associations
	if err = deleteUserACMs(tx, user); err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache error deleting existing user acms", zap.Error(err))
		return err
	}
	// 2. Get the matching ids
	var acmids []int64
	if acmids, err = getACMIDsValidForUser(tx, user); err != nil {
		tx.Rollback()
		dao.GetLogger().Error("rebuildUserACMCache had error getting list of matching ids", zap.Error(err))
		return err
	}
	// 3. With each id, add association to the user
	for idx, acmid := range acmids {
		if err := insertUserACM(tx, user, acmid); err != nil {
			tx.Rollback()
			dao.GetLogger().Error("rebuildUserACMCache error inserting useracm", zap.Int("index", idx), zap.Error(err))
			return err
		}
	}
	// 4. Done caching
	useraocache.IsCaching = false
	if err = updateUserAOCache(tx, useraocache); err != nil {
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

func getACMIDsValidForUser(tx *sqlx.Tx, user models.ODUser) ([]int64, error) {
	var acmids []int64
	snippets, err := getUserSnippets(tx, user)
	if err != nil {
		return acmids, err
	}
	acmids, err = getACMIDsValidForUserBySnippets(tx, snippets)
	return acmids, err
}

func getUserSnippets(tx *sqlx.Tx, user models.ODUser) (*acm.ODriveRawSnippetFields, error) {
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

func getACMIDsValidForUserBySnippets(tx *sqlx.Tx, snippets *acm.ODriveRawSnippetFields) ([]int64, error) {
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
	if err := tx.Select(&acmids, sql); err != nil {
		return acmids, err
	}
	return acmids, nil
}

func deleteUserACMs(tx *sqlx.Tx, user models.ODUser) error {
	sql := `delete from useracm where userid = ?`
	stmt, err := tx.Preparex(sql)
	if err != nil {
		return fmt.Errorf("deleteUserACMs error preparing delete statement, %s", err.Error())
	}
	result, err := stmt.Exec(user.ID)
	if err != nil {
		return fmt.Errorf("deleteUserACMs error executing delete statement, %s", err.Error())
	}
	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleteUserACMs error determining rows affected, %s", err.Error())
	}
	return nil
}

func doesUserHaveACM(tx *sqlx.Tx, user models.ODUser, acmID int64) (bool, error) {
	stmt, err := tx.Preparex(`select id from useracm where userid = ? and acmid = ?`)
	if err != nil {
		return false, fmt.Errorf("doesUserHaveACM error preparing select statement, %s", err.Error())
	}
	var useracmid []int64
	if err = stmt.Select(&useracmid, user.ID, acmID); err != nil {
		if err != sql.ErrNoRows {
			return false, fmt.Errorf("doesUserHaveACM error executing select statement, %s", err.Error())
		}
		return false, nil
	}
	return len(useracmid) > 0, nil
}

func insertUserACM(tx *sqlx.Tx, user models.ODUser, acmID int64) error {
	var hasACM bool
	var err error
	if hasACM, err = doesUserHaveACM(tx, user, acmID); err != nil {
		return fmt.Errorf("insertUserACM error checking if user has acm, %s", err.Error())
	}
	if hasACM {
		// nothing to do, already present, dont create dupes
		return nil
	}
	stmt, err := tx.Preparex(`insert useracm set userid = ?, acmid = ?`)
	if err != nil {
		return fmt.Errorf("insertUserACM error preparing add statement, %s", err.Error())
	}
	result, err := stmt.Exec(user.ID, acmID)
	if err != nil {
		return fmt.Errorf("insertUserACM error executing add statement, %s, acmID = %d", err.Error(), acmID)
	}
	ra, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("insertUserACM error determining rows affected, %s", err.Error())
	}
	if ra == 0 {
		return fmt.Errorf("insertUserACM did not affect any rows, %s", err.Error())
	}
	log.Printf("associating user %s to acm %d", hex.EncodeToString(user.ID), acmID)
	return nil
}

type acmkv struct {
	Key   int64 `db:"acmkey"`
	Value int64 `db:"acmvalue"`
}

func insertAssociationOfACMToModifiedByIfValid(dao *DataAccessLayer, object models.ODObject) error {
	var users []models.ODUser
	var stmt *sqlx.Stmt
	var err error
	var sql string
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		return err
	}
	sql, err = buildUsersValidForACMByIDSQL(tx, object.ACMID)
	if err != nil {
		tx.Rollback()
		return err
	}
	// tack on the user
	sql += " and u.distinguishedName = '" + MySQLSafeString(object.ModifiedBy) + "'"

	if stmt, err = tx.Preparex(sql); err != nil {
		tx.Rollback()
		return err
	}
	if err = stmt.Select(&users); err != nil {
		tx.Rollback()
		return err
	}
	if len(users) == 1 {
		user := models.ODUser{}
		user.ID = users[0].ID
		if err = insertUserACM(tx, user, object.ACMID); err != nil {
			tx.Rollback()
			return err
		}
		tx.Commit()
		return nil
	}
	tx.Rollback()
	return fmt.Errorf("expected one user for associating to acm, got %d", len(users))
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

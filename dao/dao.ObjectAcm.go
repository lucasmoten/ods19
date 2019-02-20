package dao

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/utils"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var acmFieldsRegex = regexp.MustCompile(`(^f_.*|^dissem_countries$)`)

func getOverallFlattenedACM(acmMap map[string]interface{}) string {
	var flattenedACM string

	// build sorted key list
	alphaAcmKeys := make([]string, len(acmMap))
	ak := 0
	for acmKeyName := range acmMap {
		alphaAcmKeys[ak] = acmKeyName
		ak++
	}
	sort.Strings(alphaAcmKeys)

	fieldOutputCounter := 0
	// iterate keys sorted by name
	for _, acmKeyName := range alphaAcmKeys {
		if acmFieldsRegex.MatchString(acmKeyName) {
			// dont prefix with semicolon, but do use to divide fields
			if fieldOutputCounter > 0 {
				flattenedACM += ";"
			}
			fieldOutputCounter++
			// add the key name for the current field
			flattenedACM += acmKeyName + "="
			// get value from the map
			mapValue := acmMap[acmKeyName]
			// convert to an interface array
			interfaceValue := mapValue.([]interface{})
			// get all values in the array and convert to a string array
			alphaAcmValues := make([]string, len(interfaceValue))
			av := 0
			for _, interfaceElement := range interfaceValue {
				if strings.Compare(reflect.TypeOf(interfaceElement).Kind().String(), "string") == 0 {
					interfaceStringValue := interfaceElement.(string)
					// only add non empty values
					if len(interfaceStringValue) > 0 {
						alphaAcmValues[av] = interfaceElement.(string)
						av++
					}
				}
			}
			// sort the values
			sort.Strings(alphaAcmValues)
			flattenedACM += strings.Join(alphaAcmValues, ",")
		}
	}

	return flattenedACM
}

func setObjectACM2ForObjectInTransaction(tx *sqlx.Tx, dao *DataAccessLayer, object *models.ODObject) (bool, error) {
	acmInterface, err := utils.UnmarshalStringToInterface(object.RawAcm.String)
	if err != nil {
		return false, err
	}
	acmMap, ok := acmInterface.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("Unable to convert ACM to map")
	}
	overallFlattenedACM := getOverallFlattenedACM(acmMap)
	acm, acmCreated, err := getAcm2ByNameInTransaction(tx, dao, overallFlattenedACM, true)
	if err != nil {
		return false, err
	}
	object.ACMID = acm.ID

	// If just created the ACM, parse through the map adding its parts
	if acmCreated {
		// Iterate over keys presented in the map
		for acmKeyName, mapValue := range acmMap {
			// If its a flattened value, then we care about it
			if acmFieldsRegex.MatchString(acmKeyName) {
				// Get Id for this Key, adding if Necessary
				acmKey, err := getAcmKey2ByName(dao, acmKeyName, true)
				if err != nil {
					return false, err
				}
				// Convert values to a string array
				var acmValues []string
				if mapValue != nil {
					interfaceValue := mapValue.([]interface{})
					for _, interfaceElement := range interfaceValue {
						if strings.Compare(reflect.TypeOf(interfaceElement).Kind().String(), "string") == 0 {
							newValue := interfaceElement.(string)
							if len(strings.TrimSpace(newValue)) == 0 {
								continue
							}
							found := false
							for _, existingValue := range acmValues {
								if strings.Compare(existingValue, newValue) == 0 {
									found = true
									break
								}
							}
							if !found {
								acmValues = append(acmValues, interfaceElement.(string))
							}
						}
					}
				}
				// Iterate over values presented in map
				for _, acmValueName := range acmValues {
					// Skip this entry if its empty
					if len(strings.TrimSpace(acmValueName)) == 0 {
						continue
					}
					// Get Id for this Value, adding if Necessary
					acmValue, err := getAcmValue2ByName(dao, acmValueName, true)
					if err != nil {
						return false, err
					}
					// Insert relationship of acm key and value as an acm part on the acm
					err = createAcmPart2ForACMInTransaction(tx, acm, acmKey, acmValue)
					if err != nil {
						return false, err
					}
				}
			}
		}
	}

	return acmCreated, nil
}

func getAcm2ByNameInTransaction(tx *sqlx.Tx, dao *DataAccessLayer, namedValue string, addIfMissing bool) (models.ODAcm2, bool, error) {
	created := false
	var result models.ODAcm2
	logger := dao.GetLogger()
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	retryOnErrorMessageContains := []string{"Duplicate entry", "Deadlock", "Lock wait timeout exceeded", sql.ErrNoRows.Error()}
	stmt := `select id, sha256hash, flattenedacm from acm2 where flattenedacm = ?`
	err := tx.Get(&result, stmt, namedValue)
	for retryCounter > 0 && err != nil && containsAny(err.Error(), retryOnErrorMessageContains) {
		retryCounter--
		if err == sql.ErrNoRows && addIfMissing {
			result.FlattenedACM = namedValue
			shabytes := sha256.Sum256([]byte(namedValue))
			result.SHA256Hash = fmt.Sprintf("%x", shabytes)
			var acmID int64
			if acmID, err = createAcm2InTransaction(tx, &result); err != nil {
				time.Sleep(time.Duration(retryDelay) * time.Millisecond)
				continue
			}
			result.ID = acmID
			created = true
			break
		}
		logger.Debug("restarting transaction for getAcm2ByNameInTransaction", zap.String("retryReason", firstMatch(err.Error(), retryOnErrorMessageContains)), zap.Int64("retryCounter", retryCounter))
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		err = tx.Get(&result, stmt, namedValue)
	}
	if err != nil {
		logger.Error("error in getAcm2ByNameInTransaction", zap.Error(err))
	}
	return result, created, err
}

func createAcm2InTransaction(tx *sqlx.Tx, theType *models.ODAcm2) (int64, error) {
	var newID int64
	stmt, err := tx.Preparex(`insert acm2 set sha256hash = ?, flattenedacm = ?`)
	if err != nil {
		return newID, fmt.Errorf("createAcm2InTransaction error preparing add statement, %s", err.Error())
	}
	defer stmt.Close()
	result, err := stmt.Exec(theType.SHA256Hash, theType.FlattenedACM)
	if err != nil {
		return newID, fmt.Errorf("createAcm2InTransaction error executing add statement, %s", err.Error())
	}
	theType.ID, err = result.LastInsertId()
	newID = theType.ID
	if err != nil {
		return newID, fmt.Errorf("createAcm2InTransaction error getting last inserted id, %s", err.Error())
	}
	return newID, nil
}

func getAcmKey2ByName(dao *DataAccessLayer, namedValue string, addIfMissing bool) (models.ODAcmKey2, error) {
	var result models.ODAcmKey2
	logger := dao.GetLogger()
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		logger.Error("could not begin transaction", zap.Error(err))
		return result, err
	}
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	retryOnErrorMessageContains := []string{"Duplicate entry", "Deadlock", "Lock wait timeout exceeded", sql.ErrNoRows.Error()}
	stmt := `select id, name from acmkey2 where name = ?`
	err = tx.Get(&result, stmt, namedValue)
	for retryCounter > 0 && err != nil && containsAny(err.Error(), retryOnErrorMessageContains) {
		logger.Debug("restarting transaction for getAcmKey2ByName", zap.String("retryReason", firstMatch(err.Error(), retryOnErrorMessageContains)), zap.Int64("retryCounter", retryCounter))
		tx.Rollback()
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		retryCounter--
		tx, err = dao.MetadataDB.Beginx()
		if err != nil {
			logger.Error("could not begin transaction", zap.Error(err))
			return result, err
		}
		err = tx.Get(&result, stmt, namedValue)
		if err != nil && err == sql.ErrNoRows && addIfMissing {
			result.Name = namedValue
			if err = createAcmKey2InTransaction(tx, &result); err != nil {
				continue
			}
		}
	}
	if err != nil {
		logger.Error("error in getAcmKey2ByName", zap.Error(err))
		tx.Rollback()
		return result, fmt.Errorf("error creating acm key when missing: %s", err.Error())
	}
	tx.Commit()
	return result, err
}

func createAcmKey2InTransaction(tx *sqlx.Tx, theType *models.ODAcmKey2) error {
	stmt, err := tx.Preparex(`insert acmkey2 set name = ?`)
	if err != nil {
		return fmt.Errorf("createAcmKey2InTransaction error preparing add statement, %s", err.Error())
	}
	defer stmt.Close()
	result, err := stmt.Exec(theType.Name)
	if err != nil {
		return fmt.Errorf("createAcmKey2InTransaction error executing add statement, %s", err.Error())
	}
	theType.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("createAcmKey2InTransaction error getting last inserted id, %s", err.Error())
	}
	return nil
}

func getAcmValue2ByName(dao *DataAccessLayer, namedValue string, addIfMissing bool) (models.ODAcmValue2, error) {
	var result models.ODAcmValue2
	logger := dao.GetLogger()
	tx, err := dao.MetadataDB.Beginx()
	retryCounter := dao.DeadlockRetryCounter
	retryDelay := dao.DeadlockRetryDelay
	retryOnErrorMessageContains := []string{"Duplicate entry", "Deadlock", "Lock wait timeout exceeded", sql.ErrNoRows.Error()}
	stmt := `select id, name from acmvalue2 where name = ?`
	err = tx.Get(&result, stmt, namedValue)
	for retryCounter > 0 && err != nil && containsAny(err.Error(), retryOnErrorMessageContains) {
		logger.Debug("restarting transaction for getAcmValue2ByName", zap.String("retryReason", firstMatch(err.Error(), retryOnErrorMessageContains)), zap.Int64("retryCounter", retryCounter))
		time.Sleep(time.Duration(retryDelay) * time.Millisecond)
		retryCounter--
		if err = tx.Get(&result, stmt, namedValue); err != nil {
			if err.Error() == sql.ErrNoRows.Error() && addIfMissing {
				result.Name = namedValue
				if err = createAcmValue2InTransaction(tx, &result); err != nil {
					continue
				}
			} else {
				logger.Error("error retrieving value", zap.Error(err))
			}
		}
	}
	if err != nil {
		logger.Error("error in getAcmValue2ByName", zap.Error(err))
		tx.Rollback()
		return result, fmt.Errorf("error creating acm value when missing value %s: %s", namedValue, err.Error())
	}
	tx.Commit()
	return result, err
}

func createAcmValue2InTransaction(tx *sqlx.Tx, theType *models.ODAcmValue2) error {
	stmt, err := tx.Preparex(`insert acmvalue2 set name = ?`)
	if err != nil {
		return fmt.Errorf("createAcmValue2InTransaction error preparing add statement, %s", err.Error())
	}
	defer stmt.Close()
	result, err := stmt.Exec(theType.Name)
	if err != nil {
		return fmt.Errorf("createAcmValue2InTransaction error executing add statement, %s", err.Error())
	}
	theType.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("createAcmValue2InTransaction error getting last inserted id, %s", err.Error())
	}
	return nil
}

func createAcmPart2ForACMInTransaction(tx *sqlx.Tx, acm models.ODAcm2, acmKey models.ODAcmKey2, acmValue models.ODAcmValue2) error {
	stmt, err := tx.Preparex(`insert acmpart2 set acmid = ?, acmkeyid = ?, acmvalueid = ?`)
	if err != nil {
		return fmt.Errorf("createAcmPart2ForACMInTransaction error preparing add statement, %s", err.Error())
	}
	defer stmt.Close()
	result, err := stmt.Exec(acm.ID, acmKey.ID, acmValue.ID)
	if err != nil {
		return fmt.Errorf("createAcmPart2ForACMInTransaction error executing add statement, %s", err.Error())
	}
	if _, err = result.LastInsertId(); err != nil {
		return fmt.Errorf("createAcmPart2ForACMInTransaction error getting last inserted id, %s", err.Error())
	}
	return nil
}

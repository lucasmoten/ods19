package dao

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/utils"
	"github.com/jmoiron/sqlx"
)

func setObjectACMForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject, isnew bool) error {

	// Convert serialized string to interface
	acmInterface, err := utils.UnmarshalStringToInterface(object.RawAcm.String)
	if err != nil {
		return err
	}
	// Convert interface to map
	acmMap, ok := acmInterface.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to convert ACM to map")
	}
	// Initialize Overall Flattened ACM
	overallFlattenedACM := getOverallFlattenedACM(acmMap)
	//log.Printf(overallFlattenedACM)
	acm, acmCreated, err := getAcmByNameInTransaction(tx, overallFlattenedACM, true, object.ModifiedBy)
	if err != nil {
		return err
	}
	// Insert relationship between ACM and Object
	err = setACMForObjectInTransaction(tx, object, &acm, isnew)
	if err != nil {
		return err
	}
	// If just created the ACM, parse through the map adding its parts
	if acmCreated {
		// Iterate over keys presented in the map
		for acmKeyName, mapValue := range acmMap {
			// If its a flattened value, then we care about it // TODO: Make this a regular expression or configurable?
			// All f_* fields, and also dissem_countries
			if strings.HasPrefix(acmKeyName, "f_") || strings.HasPrefix(acmKeyName, "dissem_countries") {
				// Get Id for this Key, adding if Necessary
				acmKey, err := getAcmKeyByNameInTransaction(tx, acmKeyName, true, object.ModifiedBy)
				if err != nil {
					return err
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
					acmValue, err := getAcmValueByNameInTransaction(tx, acmValueName, true, object.ModifiedBy)
					if err != nil {
						return err
					}
					// Insert relationship of acm key and value as an acm part on the acm
					err = createAcmPartForACMInTransaction(tx, acm, acmKey, acmValue)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func setACMForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject, acm *models.ODAcm, isnew bool) error {
	if isnew {
		// Object was just added, so insert this record
		addStatement, err := tx.Preparex(`insert objectacm set 
              createdBy = ?
            , objectId = ?
            , acmId = ?
        `)
		if err != nil {
			return fmt.Errorf("setACMForObjectInTransaction error preparing add statement when new, %s", err.Error())
		}
		result, err := addStatement.Exec(object.CreatedBy, object.ID, acm.ID)
		if err != nil {
			return fmt.Errorf("setACMForObjectInTransaction error executing add statement when new, %s", err.Error())
		}
		rowCount, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("setACMForObjectInTransaction error checking rows affected when new, %s", err.Error())
		}
		if rowCount < 1 {
			return fmt.Errorf("setACMForObjectInTransaction there was less than one row affected when new")
		}
	} else {

		// First check if we really need to update. ACMs can change, but the
		// value they are flattened to can remain the same. A changed ACM will
		// bring us down this codepath, but the associated acm in the acm table
		// (which is flattened/normalized) will NOT be different, and this will
		// cause a trigger error unless we do this check.
		checkExisting := `select acmId from objectacm where objectid = ? and isDeleted = 0 limit 1`
		var acmID []byte
		row := tx.QueryRow(checkExisting, object.ID)
		row.Scan(&acmID)

		stringIDOld := hex.EncodeToString(acm.ID)
		stringIDNew := hex.EncodeToString(acmID)
		if strings.Compare(stringIDOld, stringIDNew) == 0 {
			return nil
		}

		// Object already existed, so updating...
		updateStatement, err := tx.Preparex(`update objectacm set 
            modifiedBy = ?,
            acmId = ?
            where objectId = ? and isdeleted = 0
        `)
		if err != nil {
			return fmt.Errorf("setACMForObjectInTransaction error preparing update statement when changing acm, %s", err.Error())
		}
		//log.Println("acm.ID = %s", hex.EncodeToString(acm.ID))
		result, err := updateStatement.Exec(object.ModifiedBy, acm.ID, object.ID)
		if err != nil {
			return fmt.Errorf("setACMForObjectInTransaction error executing update statement when changing acm, %s", err.Error())
		}
		rowCount, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("setACMForObjectInTransaction error checking rows affected when changing acm, %s", err.Error())
		}
		if rowCount < 1 {
			return fmt.Errorf("setACMForObjectInTransaction there was less than one row affected when changing acm to id %s with name %s for object %s", hex.EncodeToString(acm.ID), acm.Name, hex.EncodeToString(object.ID))
		}
	}
	return nil
}

func createAcmPartForACMInTransaction(tx *sqlx.Tx, acm models.ODAcm, acmKey models.ODAcmKey, acmValue models.ODAcmValue) error {
	addStatement, err := tx.Preparex(`insert acmpart set 
        createdBy = ?, 
        acmId = ?,
        acmKeyId = ?,
        acmValueId = ?
    `)
	if err != nil {
		return fmt.Errorf("createAcmPartForACMInTransaction error preparing add statement, %s", err.Error())
	}
	// Add it
	result, err := addStatement.Exec(acm.CreatedBy, acm.ID, acmKey.ID, acmValue.ID)
	if err != nil {
		return fmt.Errorf("createAcmPartForACMInTransaction error executing add statement, %s", err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("createAcmPartForACMInTransaction error checking rows affected, %s", err.Error())
	}
	if rowCount < 1 {
		return fmt.Errorf("createAcmPartForACMInTransaction there was less than one row affected")
	}
	return nil
}

func getAcmByNameInTransaction(tx *sqlx.Tx, namedValue string, addIfMissing bool, createdBy string) (models.ODAcm, bool, error) {
	created := false
	var result models.ODAcm
	tableName := `acm`
	// Get the ID of the acm by its name
	getStatement := `
    select 
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,name
    from ` + tableName + `
    where
        name = ?
    order by isDeleted asc, createdDate desc limit 1    
    `
	err := tx.Get(&result, getStatement, namedValue)
	if err != nil {
		if err == sql.ErrNoRows && addIfMissing {
			// Clear the error from no rows
			err = nil
			// Prepare new type
			result.Name = namedValue
			result.CreatedBy = createdBy
			// Create it
			createdResult, err := createAcmInTransaction(tx, &result)
			// Any errors? return them
			if err != nil {
				// Empty return with error from creation
				return result, false, fmt.Errorf("Error creating "+tableName+" when missing: %s", err.Error())
			}
			// Assign created type to the return value
			result = createdResult
			created = true
		} else {
			// Some other error besides no matching rows...
			// Empty return type with error retrieving
			return result, false, fmt.Errorf("getAcmByNameInTransaction error, %s", err.Error())
		}
	}

	// Need to make sure the type retrieved isn't deleted.
	if result.IsDeleted {
		// Existing type is deleted. Should a new active type be created?
		if addIfMissing {
			// Prepare new type
			result.Name = namedValue
			result.CreatedBy = createdBy
			// Create it
			createdResult, err := createAcmInTransaction(tx, &result)
			// Any errors? return them
			if err != nil {
				// Reinitialize result first since it may be dirty at this
				// phase and caller may accidentally use if not properly
				// checking errors
				result = models.ODAcm{}
				return result, false, fmt.Errorf("Error recreating "+tableName+" when previous was deleted: %s", err.Error())
			}
			// Assign created type to the return value
			result = createdResult
			created = true
		}
	}

	// Return response
	return result, created, err
}

func createAcmInTransaction(tx *sqlx.Tx, theType *models.ODAcm) (models.ODAcm, error) {
	var dbType models.ODAcm
	tableName := `acm`
	addStatement, err := tx.Preparex(`insert ` + tableName + ` set 
        createdBy = ?
        ,name = ?
    `)
	if err != nil {
		return dbType, fmt.Errorf("createAcmInTransaction error preparing add statement, %s", err.Error())
	}
	// Add it
	result, err := addStatement.Exec(theType.CreatedBy, theType.Name)
	if err != nil {
		return dbType, fmt.Errorf("createAcmInTransaction error executing add statement, %s", err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		return dbType, fmt.Errorf("createAcmInTransaction error checking rows affected, %s", err.Error())
	}
	if rowCount < 1 {
		return dbType, fmt.Errorf("createAcmInTransaction there was less than one row affected")
	}
	// Get the ID of the newly created type and assign to passed in objectType
	getStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,name
    from ` + tableName + ` 
    where 
        createdBy = ?
        and name = ? 
        and isdeleted = 0 
    order by createdDate desc limit 1`
	err = tx.Get(&dbType, getStatement, theType.CreatedBy, theType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbType, fmt.Errorf("createAcmInTransaction type was not found even after just adding it!, %s", err.Error())
		}
		return dbType, fmt.Errorf("createAcmInTransaction error getting newly added type, %s", err.Error())
	}
	theType = &dbType
	return dbType, nil
}

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
		if strings.HasPrefix(acmKeyName, "f_") || strings.HasPrefix(acmKeyName, "dissem_countries") {
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
			// iterate values to append them to the flattened acm
			for av2, acmValue := range alphaAcmValues {
				if av2 <= av {
					// comma delimit the values
					if av2 > 0 {
						flattenedACM += ","
					}
					flattenedACM += acmValue
				}
			}
		}
	}

	return flattenedACM
}

// adds a record in acmpart linking acm to acmkey and acmvalue if an active one does't already exist
func createACMPartIfNotExists(tx *sqlx.Tx, acm *models.ODAcm, acmKey models.ODAcmKey, acmValue models.ODAcmValue) error {
	var dbRecord models.ODAcmPart
	getStatement := `
    select 
        ap.id 'id'
        ,ap.createdDate 'createdDate'
        ,ap.createdBy 'createdBy'
        ,ap.modifiedDate 'modifiedDate'
        ,ap.modifiedBy 'modifiedBy'
        ,ap.isDeleted 'isDeleted'
        ,ap.deletedDate 'deletedDate'
        ,ap.deletedBy 'deletedBy'
        ,ap.acmId 'acmId'
        ,ap.acmKeyId 'acmKeyId'
        ,ak.name 'acmKeyName'
        ,ap.acmValueId 'acmValueId'
        ,av.name 'acmValueName'
    from
        acmpart ap
        inner join acmkey ak on ap.acmKeyId = ak.id
        inner join acmvalue av on ap.acmValueId = av.id
    where
        ap.isDeleted = 0
        and ap.acmId = ?
        and ap.acmKeyId = ?
        and ap.acmValueId = ?
    `
	err := tx.Get(&dbRecord, getStatement, acm.ID, acmKey.ID, acmValue.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Clear the error from no rows
			err = nil
			// Prepare to add it
			addStatement, err := tx.Preparex(`insert acmpart set 
                createdBy = ?
                ,acmId = ?
                ,acmKeyId = ?
                ,acmValueId = ?
            `)
			if err != nil {
				return fmt.Errorf("createACMPartIfNotExists Error preparing add statement, %s", err.Error())
			}
			// Add it
			result, err := addStatement.Exec(acm.ModifiedBy, acm.ID, acmKey.ID, acmValue.ID)
			if err != nil {
				return fmt.Errorf("createACMPartIfNotExists Error executing add statement, %s", err.Error())
			}
			err = addStatement.Close()
			if err != nil {
				return fmt.Errorf("createACMPartIfNotExists Error closing addStatement, %s", err.Error())
			}
			// Check that a row was affected
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("createACMPartIfNotExists Error checking result for rows affected, %s", err.Error())
			}
			if rowsAffected <= 0 {
				return fmt.Errorf("createACMPartIfNotExists inserted but no rows affected")
			}
		} else {
			// Some other error besides no matching rows...
			return err
		}
	}
	return nil
}

func getAcmKeyByNameInTransaction(tx *sqlx.Tx, namedValue string, addIfMissing bool, createdBy string) (models.ODAcmKey, error) {

	var result models.ODAcmKey
	tableName := `acmkey`
	// Get the ID of the acmkey by its name
	getStatement := `
    select 
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,name
    from ` + tableName + `
    where
        name = ?
    order by isDeleted asc, createdDate desc limit 1    
    `
	err := tx.Get(&result, getStatement, namedValue)
	if err != nil {
		if err == sql.ErrNoRows && addIfMissing {
			// Clear the error from no rows
			err = nil
			// Prepare new type
			result.Name = namedValue
			result.CreatedBy = createdBy
			// Create it
			createdResult, err := createAcmKeyInTransaction(tx, &result)
			// Any errors? return them
			if err != nil {
				// Empty return with error from creation
				return result, fmt.Errorf("Error creating "+tableName+" when missing: %s", err.Error())
			}
			// Assign created type to the return value
			result = createdResult
		} else {
			// Some other error besides no matching rows...
			// Empty return type with error retrieving
			return result, fmt.Errorf("getAcmKeyByNameInTransaction error, %s", err.Error())
		}
	}

	// Need to make sure the type retrieved isn't deleted.
	if result.IsDeleted {
		// Existing type is deleted. Should a new active type be created?
		if addIfMissing {
			// Prepare new type
			result.Name = namedValue
			result.CreatedBy = createdBy
			// Create it
			createdResult, err := createAcmKeyInTransaction(tx, &result)
			// Any errors? return them
			if err != nil {
				// Reinitialize result first since it may be dirty at this
				// phase and caller may accidentally use if not properly
				// checking errors
				result = models.ODAcmKey{}
				return result, fmt.Errorf("Error recreating "+tableName+" when previous was deleted: %s", err.Error())
			}
			// Assign created type to the return value
			result = createdResult
		}
	}

	// Return response
	return result, err
}

func createAcmKeyInTransaction(tx *sqlx.Tx, theType *models.ODAcmKey) (models.ODAcmKey, error) {
	var dbType models.ODAcmKey
	tableName := `acmkey`
	addStatement, err := tx.Preparex(`insert ` + tableName + ` set 
        createdBy = ?
        ,name = ?
    `)
	if err != nil {
		return dbType, fmt.Errorf("createAcmKeyInTransaction error preparing add statement, %s", err.Error())
	}
	// Add it
	result, err := addStatement.Exec(theType.CreatedBy, theType.Name)
	if err != nil {
		return dbType, fmt.Errorf("createAcmKeyInTransaction error executing add statement, %s", err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		return dbType, fmt.Errorf("createAcmKeyInTransaction error checking rows affected, %s", err.Error())
	}
	if rowCount < 1 {
		return dbType, fmt.Errorf("createAcmKeyInTransaction there was less than one row affected")
	}
	// Get the ID of the newly created type and assign to passed in objectType
	getStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,name
    from ` + tableName + ` 
    where 
        createdBy = ?
        and name = ? 
        and isdeleted = 0 
    order by createdDate desc limit 1`
	err = tx.Get(&dbType, getStatement, theType.CreatedBy, theType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbType, fmt.Errorf("createAcmKeyInTransaction type was not found even after just adding it!, %s", err.Error())
		}
		return dbType, fmt.Errorf("createAcmKeyInTransaction error getting newly added type, %s", err.Error())
	}
	theType = &dbType
	return dbType, nil
}

func getAcmValueByNameInTransaction(tx *sqlx.Tx, namedValue string, addIfMissing bool, createdBy string) (models.ODAcmValue, error) {

	var result models.ODAcmValue
	tableName := `acmvalue`
	// Get the ID of the acmkey by its name
	getStatement := `
    select 
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,name
    from ` + tableName + `
    where
        name = ?
    order by isDeleted asc, createdDate desc limit 1    
    `
	err := tx.Get(&result, getStatement, namedValue)
	if err != nil {
		if err == sql.ErrNoRows && addIfMissing {
			// Clear the error from no rows
			err = nil
			// Prepare new type
			result.Name = namedValue
			result.CreatedBy = createdBy
			// Create it
			createdResult, err := createAcmValueInTransaction(tx, &result)
			// Any errors? return them
			if err != nil {
				// Empty return with error from creation
				return result, fmt.Errorf("Error creating "+tableName+" when missing: %s", err.Error())
			}
			// Assign created type to the return value
			result = createdResult
		} else {
			// Some other error besides no matching rows...
			// Empty return type with error retrieving
			return result, fmt.Errorf("getAcmValueByNameInTransaction error, %s", err.Error())
		}
	}

	// Need to make sure the type retrieved isn't deleted.
	if result.IsDeleted {
		// Existing type is deleted. Should a new active type be created?
		if addIfMissing {
			// Prepare new type
			result.Name = namedValue
			result.CreatedBy = createdBy
			// Create it
			createdResult, err := createAcmValueInTransaction(tx, &result)
			// Any errors? return them
			if err != nil {
				// Reinitialize result first since it may be dirty at this
				// phase and caller may accidentally use if not properly
				// checking errors
				result = models.ODAcmValue{}
				return result, fmt.Errorf("Error recreating "+tableName+" when previous was deleted: %s", err.Error())
			}
			// Assign created type to the return value
			result = createdResult
		}
	}

	// Return response
	return result, err
}

func createAcmValueInTransaction(tx *sqlx.Tx, theType *models.ODAcmValue) (models.ODAcmValue, error) {
	var dbType models.ODAcmValue
	tableName := `acmvalue`
	addStatement, err := tx.Preparex(`insert ` + tableName + ` set 
        createdBy = ?
        ,name = ?
    `)
	if err != nil {
		return dbType, fmt.Errorf("createAcmValueInTransaction error preparing add statement, %s", err.Error())
	}
	// Add it
	result, err := addStatement.Exec(theType.CreatedBy, theType.Name)
	if err != nil {
		return dbType, fmt.Errorf("createAcmValueInTransaction error executing add statement, %s", err.Error())
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if err != nil {
		return dbType, fmt.Errorf("createAcmValueInTransaction error checking rows affected, %s", err.Error())
	}
	if rowCount < 1 {
		return dbType, fmt.Errorf("createAcmValueInTransaction there was less than one row affected")
	}
	// Get the ID of the newly created type and assign to passed in objectType
	getStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,name
    from ` + tableName + ` 
    where 
        createdBy = ?
        and name = ? 
        and isdeleted = 0 
    order by createdDate desc limit 1`
	err = tx.Get(&dbType, getStatement, theType.CreatedBy, theType.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbType, fmt.Errorf("createAcmValueInTransaction type was not found even after just adding it!, %s", err.Error())
		}
		return dbType, fmt.Errorf("createAcmValueInTransaction error getting newly added type, %s", err.Error())
	}
	theType = &dbType
	return dbType, nil
}

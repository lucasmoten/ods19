package dao

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

func setObjectACMForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) error {

	// Convert serialized string to interface
	acmInterface, err := mapping.ConvertStringACMToObject(object.RawAcm.String)
	if err != nil {
		return err
	}
	// Convert interface to map
	acmMap, ok := acmInterface.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to convert ACM to map")
	}
	// Get existing keys for this object
	dbAcmKeys, err := getAcmKeysForObjectInTransaction(tx, object)
	if err != nil {
		return fmt.Errorf("Unable to retrieve acm keys %s", err.Error())
	}
	// Iterate over keys presented in the map
	for acmKeyName, mapValue := range acmMap {
		// If its a flattened value, then we care about it // TODO: Make this a regular expression or configurable?
		// All f_* fields, except f_share, and also dissem_countries
		if (strings.HasPrefix(acmKeyName, "f_") || strings.HasPrefix(acmKeyName, "dissem_countries")) && (!strings.HasPrefix(acmKeyName, "f_share")) {
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
						acmValues = append(acmValues, interfaceElement.(string))
					}
				}
			}
			// Get existing values for this object and acm key
			dbAcmValues, err := getAcmKeyValuesForObjectInTransaction(tx, object, acmKey)
			if err != nil {
				return fmt.Errorf("Unable to retrieve acm values %s", err.Error())
			}
			// Iterate over values presented in map
			for _, acmValueName := range acmValues {
				// Get Id for this Value, adding if Necessary
				acmValue, err := getAcmValueByNameInTransaction(tx, acmValueName, true, object.ModifiedBy)
				if err != nil {
					return err
				}
				// Insert relationsip of acm key and value to object if not already exist
				err = createObjectACMIfNotExists(tx, object, acmKey, acmValue)
				if err != nil {
					return err
				}
				// Iterate over previously existing db values for the key
				if len(dbAcmValues) > 0 {
					for dbAcmValuePos, dbAcmValue := range dbAcmValues {
						if strings.Compare(dbAcmValue.Name, acmValueName) == 0 {
							// found, remove from this slice
							dbAcmValues = append(dbAcmValues[:dbAcmValuePos], dbAcmValues[dbAcmValuePos+1:]...)
							// and bail from loop
							break
						}
					}
				}
			}
			// Anything remaining in dbAcmValues is no longer needed for this object and acm key
			if len(dbAcmValues) > 0 {
				for _, dbAcmValue := range dbAcmValues {
					err = removeAcmKeyValueForObjectInTransaction(tx, object, acmKey, dbAcmValue)
					if err != nil {
						return err
					}
				}
			}
			// Iterate over previously existing db keys for the object
			if len(dbAcmKeys) > 0 {
				for dbAcmKeyPos, dbAcmKey := range dbAcmKeys {
					if strings.Compare(dbAcmKey.Name, acmKeyName) == 0 {
						// found, remove from this slice
						dbAcmKeys = append(dbAcmKeys[:dbAcmKeyPos], dbAcmKeys[dbAcmKeyPos+1:]...)
						// and bail from this loop
						break
					}
				}
			}
		}
	}
	// Anything remaining in dbAcmKeys is no longer needed for this object
	if len(dbAcmKeys) > 0 {
		for _, dbAcmKey := range dbAcmKeys {
			err = removeAcmKeyForObjectInTransaction(tx, object, dbAcmKey)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func removeAcmKeyForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject, acmKey models.ODAcmKey) error {
	// update statement
	updateStatement, err := tx.Preparex(`update object_acm set modifiedBy = ?, deletedBy = ?, isDeleted = 1 where objectId = ? and acmKeyId = ?`)
	if err != nil {
		return fmt.Errorf("removeAcmKeyForObjectInTransaction Preparing update statement, %s", err.Error())
	}
	// Update it
	_, err = updateStatement.Exec(object.ModifiedBy, object.DeletedBy, object.ID, acmKey.ID)
	if err != nil {
		return fmt.Errorf("removeAcmKeyForObjectInTransaction Error executing update statement, %s", err.Error())
	}
	// Close statement
	updateStatement.Close()
	// Done
	return nil
}
func removeAcmKeyValueForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject, acmKey models.ODAcmKey, acmValue models.ODAcmValue) error {
	// update statement
	updateStatement, err := tx.Preparex(`update object_acm set modifiedBy = ?, deletedBy = ?, isDeleted = 1 where objectId = ? and acmKeyId = ? and acmValueId = ?`)
	if err != nil {
		return fmt.Errorf("removeAcmKeyValueForObjectInTransaction Preparing update statement, %s", err.Error())
	}
	// Update it
	_, err = updateStatement.Exec(object.ModifiedBy, object.DeletedBy, object.ID, acmKey.ID, acmValue.ID)
	if err != nil {
		return fmt.Errorf("removeAcmKeyValueForObjectInTransaction Error executing update statement, %s", err.Error())
	}
	// Close statement
	updateStatement.Close()
	// Done
	return nil
}
func getAcmKeysForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject) ([]models.ODAcmKey, error) {
	var dbAcmKeys []models.ODAcmKey
	getStatement := `
    select distinct
         ak.id 'id'
        ,ak.createdDate 'createdDate'
        ,ak.createdBy 'createdBy'
        ,ak.modifiedDate 'modifiedDate'
        ,ak.modifiedBy 'modifiedBy'
        ,ak.isDeleted 'isDeleted'
        ,ak.deletedDate 'deletedDate'
        ,ak.deletedBy 'deletedBy'
        ,ak.name 'name'
    from
        object_acm oa
        inner join acmkey ak on oa.acmKeyId = ak.id
    where
        oa.isDeleted = 0
        and oa.objectId = ?        
    `
	err := tx.Select(&dbAcmKeys, getStatement, object.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbAcmKeys, nil
		}
		return dbAcmKeys, err
	}
	return dbAcmKeys, nil
}
func getAcmKeyValuesForObjectInTransaction(tx *sqlx.Tx, object *models.ODObject, acmKey models.ODAcmKey) ([]models.ODAcmValue, error) {
	var dbAcmValues []models.ODAcmValue
	getStatement := `
    select 
         av.id 'id'
        ,av.createdDate 'createdDate'
        ,av.createdBy 'createdBy'
        ,av.modifiedDate 'modifiedDate'
        ,av.modifiedBy 'modifiedBy'
        ,av.isDeleted 'isDeleted'
        ,av.deletedDate 'deletedDate'
        ,av.deletedBy 'deletedBy'
        ,av.name 'name'
    from
        object_acm oa
        inner join acmvalue av on oa.acmValueId = av.id
    where
        oa.isDeleted = 0
        and oa.objectId = ?
        and oa.acmKeyId = ?        
    `
	err := tx.Select(&dbAcmValues, getStatement, object.ID, acmKey.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return dbAcmValues, nil
		}
		return dbAcmValues, err
	}
	return dbAcmValues, nil
}

// adds a record in object_acm linking object to acmkey and acmvalue if an active one does't already exist
func createObjectACMIfNotExists(tx *sqlx.Tx, object *models.ODObject, acmKey models.ODAcmKey, acmValue models.ODAcmValue) error {
	var dbRecord models.ODObjectAcm
	getStatement := `
    select 
        oa.id 'id'
        ,oa.createdDate 'createdDate'
        ,oa.createdBy 'createdBy'
        ,oa.modifiedDate 'modifiedDate'
        ,oa.modifiedBy 'modifiedBy'
        ,oa.isDeleted 'isDeleted'
        ,oa.deletedDate 'deletedDate'
        ,oa.deletedBy 'deletedBy'
        ,oa.objectId 'objectId'
        ,oa.acmKeyId 'acmKeyId'
        ,ak.name 'acmKeyName'
        ,oa.acmValueId 'acmValueId'
        ,av.name 'acmValueName'
    from
        object_acm oa
        inner join acmkey ak on oa.acmKeyId = ak.id
        inner join acmvalue av on oa.acmValueId = av.id
    where
        oa.isDeleted = 0
        and oa.objectId = ?
        and oa.acmKeyId = ?
        and oa.acmValueId = ?
    `
	err := tx.Get(&dbRecord, getStatement, object.ID, acmKey.ID, acmValue.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Clear the error from no rows
			err = nil
			// Prepare to add it
			addStatement, err := tx.Preparex(`insert object_acm set 
                createdBy = ?
                ,objectId = ?
                ,acmKeyId = ?
                ,acmValueId = ?
            `)
			if err != nil {
				return fmt.Errorf("createObjectACMIfNotExists Error preparing add statement, %s", err.Error())
			}
			// Add it
			result, err := addStatement.Exec(object.ModifiedBy, object.ID, acmKey.ID, acmValue.ID)
			if err != nil {
				return fmt.Errorf("createObjectACMIfNotExists Error executing add statement, %s", err.Error())
			}
			err = addStatement.Close()
			if err != nil {
				return fmt.Errorf("createObjectACMIfNotExists Error closing addStatement, %s", err.Error())
			}
			// Check that a row was affected
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("createObjectACMIfNotExists Error checking result for rows affected, %s", err.Error())
			}
			if rowsAffected <= 0 {
				return fmt.Errorf("createObjectACMIfNotExists inserted but no rows affected")
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
		if err == sql.ErrNoRows {
			if addIfMissing {
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
			}
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
		if err == sql.ErrNoRows {
			if addIfMissing {
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
			}
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

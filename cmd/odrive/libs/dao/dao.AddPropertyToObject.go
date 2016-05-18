package dao

import (
	"errors"
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/metadata/models"
)

// AddPropertyToObject creates a new property with the provided name and value,
// and then associates that Property object to the Object indicated by ObjectID
func (dao *DataAccessLayer) AddPropertyToObject(object models.ODObject, property *models.ODProperty) (models.ODProperty, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		log.Printf("Could not begin transaction: %v", err)
		return models.ODProperty{}, err
	}
	dbProperty, err := addPropertyToObjectInTransaction(tx, object, property)
	if err != nil {
		log.Printf("Error in AddPropertyToObject: %v", err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbProperty, err
}

func addPropertyToObjectInTransaction(tx *sqlx.Tx, object models.ODObject, property *models.ODProperty) (models.ODProperty, error) {
	var dbProperty models.ODProperty

	// Setup the statement
	addPropertyStatement, err := tx.Preparex(`insert property set 
        createdby = ?
        ,name = ?
        ,propertyvalue = ?
        ,classificationpm = ?
    `)
	if err != nil {
		return dbProperty, err
	}
	// Add it
	result, err := addPropertyStatement.Exec(property.CreatedBy, property.Name, property.Value.String, property.ClassificationPM.String)
	if err != nil {
		return dbProperty, err
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		return dbProperty, errors.New("No rows added from inserting property")
	}
	addPropertyStatement.Close()
	// Get the ID of the newly created property
	var newPropertyID []byte
	getPropertyIDStatement, err := tx.Preparex(`
    select 
        id 
    from property 
    where 
        createdby = ? 
        and name = ? 
        and propertyvalue = ? 
        and classificationpm = ? 
    order by createddate desc limit 1`)
	if err != nil {
		return dbProperty, err
	}
	err = getPropertyIDStatement.QueryRowx(property.CreatedBy, property.Name, property.Value.String, property.ClassificationPM.String).Scan(&newPropertyID)
	if err != nil {
		return dbProperty, err
	}
	getPropertyIDStatement.Close()
	// Retrieve back into property
	getPropertyStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,changeCount
        ,changeToken
        ,name
        ,propertyValue
        ,classificationPM    
    from property
    where id = ?
    `
	err = tx.Get(&dbProperty, getPropertyStatement, newPropertyID)
	if err != nil {
		return dbProperty, err
	}
	*property = dbProperty
	// Add association to the object
	addObjectPropertyStatement, err := tx.Preparex(`insert object_property set 
        createdby = ?
        ,objectid = ?
        ,propertyid = ?
    `)
	if err != nil {
		return dbProperty, err
	}
	result, err = addObjectPropertyStatement.Exec(property.CreatedBy, object.ID, newPropertyID)
	if err != nil {
		return dbProperty, err
	}
	rowCount, err = result.RowsAffected()
	if rowCount < 1 {
		return dbProperty, errors.New("No rows added from inserting object_property")
	}
	addObjectPropertyStatement.Close()

	return dbProperty, nil
}

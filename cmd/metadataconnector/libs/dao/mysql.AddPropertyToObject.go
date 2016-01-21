package dao

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

// AddPropertyToObject creates a new property with the provided name and value,
// and then associates that Property object to the Object indicated by ObjectID
func AddPropertyToObject(db *sqlx.DB, createdBy string, objectID []byte, propertyName string, propertyValue string, classificationPM string) error {
	// Setup the statement
	addPropertyStatement, err := db.Prepare(`insert property set createdby = ?, name = ?, propertyvalue = ?, classificationpm = ?`)
	if err != nil {
		return err
	}
	// Add it
	result, err := addPropertyStatement.Exec(createdBy, propertyName, propertyValue, classificationPM)
	if err != nil {
		return err
	}
	// Cannot use result.LastInsertId() as our identifier is not an autoincremented int
	rowCount, err := result.RowsAffected()
	if rowCount < 1 {
		return errors.New("No rows added from inserting property")
	}
	// Get the ID of the newly created property
	var newPropertyID []byte
	getPropertyIDStatement, err := db.Prepare(`select id from property where createdby = ? and name = ? and propertyvalue = ? and classificationpm = ? order by createddate desc limit 1`)
	if err != nil {
		return err
	}
	err = getPropertyIDStatement.QueryRow(createdBy, propertyName, propertyValue, classificationPM).Scan(&newPropertyID)
	if err != nil {
		return err
	}
	// Add association to the object
	addObjectPropertyStatement, err := db.Prepare(`insert object_property set createdby = ?, objectid = ?, propertyid = ?`)
	result, err = addObjectPropertyStatement.Exec(createdBy, objectID, newPropertyID)
	if err != nil {
		return err
	}

	return nil
}

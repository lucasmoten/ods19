package dao

import (
	"errors"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// AddPropertyToObject creates a new property with the provided name and value,
// and then associates that Property object to the Object indicated by ObjectID
func AddPropertyToObject(db *sqlx.DB, createdBy string, object *models.ODObject, property *models.ODProperty) error {
	tx := db.MustBegin()
	// Setup the statement
	addPropertyStatement, err := tx.Prepare(`insert property set createdby = ?, name = ?, propertyvalue = ?, classificationpm = ?`)
	if err != nil {
		return err
	}
	// Add it
	result, err := addPropertyStatement.Exec(createdBy, property.Name, property.Value.String, property.ClassificationPM.String)
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
	getPropertyIDStatement, err := tx.Prepare(`select id from property where createdby = ? and name = ? and propertyvalue = ? and classificationpm = ? order by createddate desc limit 1`)
	if err != nil {
		return err
	}
	err = getPropertyIDStatement.QueryRow(createdBy, property.Name, property.Value.String, property.ClassificationPM.String).Scan(&newPropertyID)
	if err != nil {
		return err
	}
	// Retrieve back into property
	err = tx.Get(property, `select * from property where id = ?`, newPropertyID)
	if err != nil {
		return err
	}
	// Add association to the object
	addObjectPropertyStatement, err := tx.Prepare(`insert object_property set createdby = ?, objectid = ?, propertyid = ?`)
	if err != nil {
		return err
	}
	result, err = addObjectPropertyStatement.Exec(createdBy, object.ID, newPropertyID)
	if err != nil {
		return err
	}
	rowCount, err = result.RowsAffected()
	if rowCount < 1 {
		return errors.New("No rows added from inserting object_property")
	}
	tx.Commit()

	return nil
}

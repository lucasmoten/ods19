package dao

import (
	"errors"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/metadata/models"
)

// AddPropertyToObject creates a new property with the provided name and value,
// and then associates that Property object to the Object indicated by ObjectID
func (dao *DataAccessLayer) AddPropertyToObject(createdBy string, object *models.ODObject, property *models.ODProperty) error {
	tx := dao.MetadataDB.MustBegin()
	err := addPropertyToObjectInTransaction(tx, createdBy, object, property)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func addPropertyToObjectInTransaction(tx *sqlx.Tx, createdBy string, object *models.ODObject, property *models.ODProperty) error {
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
	addPropertyStatement.Close()
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
	getPropertyIDStatement.Close()
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
	addObjectPropertyStatement.Close()

	return nil
}

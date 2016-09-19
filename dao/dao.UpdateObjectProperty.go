package dao

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
)

// UpdateObjectProperty uses the passed in objectProperty and makes the
// appropriate sql calls to the database to validate that the token is current,
// and is not yet deleted before making changes to the property value and its
// classification portion mark
//    objectProperty.ID must be set to the property to be updated
//    objectProperty.ChangeToken must be set to the current value
//    objectProperty.ModifiedBy must be set to the user performing the operation
//    objectProperty.Value.String must be set to the new value.
func (dao *DataAccessLayer) UpdateObjectProperty(objectProperty models.ODObjectPropertyEx) error {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		dao.GetLogger().Error("Could not begin transaction", zap.String("err", err.Error()))
		return err
	}
	err = updateObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		dao.GetLogger().Error("Error in UpdateObjectProperty", zap.String("err", err.Error()))
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func updateObjectPropertyInTransaction(tx *sqlx.Tx, objectProperty models.ODObjectPropertyEx) error {
	// Pre-DB Validation
	if objectProperty.ID == nil {
		return ErrMissingID
	}
	if objectProperty.ChangeToken == "" {
		return ErrMissingChangeToken
	}
	// Fetch object property
	dbObjectProperty, err := getObjectPropertyInTransaction(tx, objectProperty)
	if err != nil {
		return err
	}
	// Check if changeToken matches
	if objectProperty.ChangeToken != dbObjectProperty.ChangeToken {
		return fmt.Errorf("ChangeToken does not match expected value %s", dbObjectProperty.ChangeToken)
	}
	// Check if deleted
	if dbObjectProperty.IsDeleted {
		// NOOP
		return nil
	}
	// Setup property
	dbObjectProperty.ModifiedBy = objectProperty.ModifiedBy
	dbObjectProperty.Value.String = objectProperty.Value.String
	dbObjectProperty.ClassificationPM.String = objectProperty.ClassificationPM.String
	updateObjectPropertyStatement, err := tx.Preparex(`update property set 
        modifiedby = ?
        ,value = ?
        ,classificationpm = ? 
    where id = ?`)
	if err != nil {
		return err
	}
	_, err = updateObjectPropertyStatement.Exec(dbObjectProperty.ModifiedBy, dbObjectProperty.Value.String, dbObjectProperty.ClassificationPM.String, dbObjectProperty.IsDeleted, dbObjectProperty.ID)
	if err != nil {
		return err
	}

	return nil
}
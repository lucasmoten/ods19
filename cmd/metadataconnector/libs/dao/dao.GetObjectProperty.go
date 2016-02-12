package dao

import "decipher.com/oduploader/metadata/models"

// GetObjectProperty return the requested property by ID.
// NOTE: Should we just pass an ID instead?
func (dao *DataAccessLayer) GetObjectProperty(objectProperty *models.ODObjectPropertyEx) (*models.ODObjectPropertyEx, error) {

	var dbObjectProperty models.ODObjectPropertyEx
	query := `select * from property where id = ?`
	err := dao.MetadataDB.Get(&dbObjectProperty, query, objectProperty.ID)
	if err != nil {
		print(err.Error())
	}
	return &dbObjectProperty, err
}

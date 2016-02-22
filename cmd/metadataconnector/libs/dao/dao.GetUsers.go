package dao

import (
	"decipher.com/oduploader/metadata/models"
	"log"
)

// GetUsers retrieves all users.
func (dao *DataAccessLayer) GetUsers() ([]models.ODUser, error) {
	// TODO this should return a User struct
	var result []models.ODUser
	getUsersStatement := `select distinguishedName, displayName, email from user`
	err := dao.MetadataDB.Select(&result, getUsersStatement)
	if err != nil {
		log.Printf("Unable to execute query %s:%v", getUsersStatement, err)
	}
	return result, err
}

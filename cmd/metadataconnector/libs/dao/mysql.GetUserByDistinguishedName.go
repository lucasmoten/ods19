package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func GetUserByDistinguishedName(db *sqlx.DB, user *models.ODUser) (*models.ODUser, error) {
	var dbUser models.ODUser
	getUserStatement := `select * from user where distinguishedName = ?`
	err := db.Get(&dbUser, getUserStatement, user.DistinguishedName)
	return &dbUser, err
}

package dao

import (
	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func GetUserByDistinguishedName(db *sqlx.DB, user *models.ODUser) error {
	getUserStatement := `select * from user where distinguishedName = ?`
	err := db.Get(&user, getUserStatement, user.DistinguishedName)
	return err
}

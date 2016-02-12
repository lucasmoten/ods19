package dao

import "decipher.com/oduploader/metadata/models"

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func (dao *DataAccessLayer) GetUserByDistinguishedName(user *models.ODUser) (*models.ODUser, error) {
	var dbUser models.ODUser
	getUserStatement := `select * from user where distinguishedName = ?`
	err := dao.MetadataDB.Get(&dbUser, getUserStatement, user.DistinguishedName)
	return &dbUser, err
}

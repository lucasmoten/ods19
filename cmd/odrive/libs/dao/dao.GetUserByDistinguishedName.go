package dao

import (
	"log"

	"database/sql"

	"decipher.com/object-drive-server/metadata/models"
	"github.com/jmoiron/sqlx"
)

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func (dao *DataAccessLayer) GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error) {
	tx, err := dao.MetadataDB.Beginx()
	if err != nil {
		log.Printf("Could not begin transaction: %v", err)
		return models.ODUser{}, err
	}
	dbUser, err := getUserByDistinguishedNameInTransaction(tx, user)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Error in GetUserByDistinguishedName: %v", err)
		}
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return dbUser, err
}

func getUserByDistinguishedNameInTransaction(tx *sqlx.Tx, user models.ODUser) (models.ODUser, error) {
	var dbUser models.ODUser
	getUserStatement := `
    select
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,changeCount
        ,changeToken
        ,distinguishedName
        ,displayName
        ,email
    from user 
    where distinguishedName = ?`
	err := tx.Get(&dbUser, getUserStatement, user.DistinguishedName)
	return dbUser, err
}

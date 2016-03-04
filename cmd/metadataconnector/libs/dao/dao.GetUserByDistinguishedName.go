package dao

import (
	"log"

	"decipher.com/oduploader/metadata/models"
	"github.com/jmoiron/sqlx"
    "database/sql"
)

// GetUserByDistinguishedName looks up user record from the database using the
// provided distinguished name
func (dao *DataAccessLayer) GetUserByDistinguishedName(user models.ODUser) (models.ODUser, error) {
	tx := dao.MetadataDB.MustBegin()
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
	getUserStatement := `select * from user where distinguishedName = ?`
	err := tx.Get(&dbUser, getUserStatement, user.DistinguishedName)
	return dbUser, err
}

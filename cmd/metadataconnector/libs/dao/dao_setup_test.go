package dao_test

import (
	"log"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

var db *sqlx.DB
var d *dao.DataAccessLayer

func init() {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	var err error
	db, err = dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err)
	}

	d = &dao.DataAccessLayer{MetadataDB: db}

	// Create users referenced by these tests
	var user *models.ODUser
	var user1 models.ODUser
	user1.DistinguishedName = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	user1.DisplayName.String = config.GetCommonName(user1.DistinguishedName)
	user1.DisplayName.Valid = true
	user1.CreatedBy = user1.DistinguishedName
	user, err = d.CreateUser(&user1)
	log.Printf("User 1 Change Count and Token: %d - %s", user.ChangeCount, user.ChangeToken)
	var user2 models.ODUser
	user2.DistinguishedName = "CN=test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	user2.DisplayName.String = config.GetCommonName(user2.DistinguishedName)
	user2.DisplayName.Valid = true
	user2.CreatedBy = user2.DistinguishedName
	user, err = d.CreateUser(&user2)
	log.Printf("User 2 Change Count and Token: %d - %s", user.ChangeCount, user.ChangeToken)
	var user10 models.ODUser
	user10.DistinguishedName = "CN=test tester10, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	user10.DisplayName.String = config.GetCommonName(user10.DistinguishedName)
	user10.DisplayName.Valid = true
	user10.CreatedBy = user10.DistinguishedName
	user, err = d.CreateUser(&user10)
	log.Printf("User 10 Change Count and Token: %d - %s", user.ChangeCount, user.ChangeToken)

}

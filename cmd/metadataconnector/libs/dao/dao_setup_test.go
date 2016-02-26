package dao_test

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

var db *sqlx.DB
var d *dao.DataAccessLayer
var usernames = make([]string, 10)

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
	user := models.ODUser{}
	//var createdUser *models.ODUser
	for i := 0; i < len(usernames); i++ {
		if i == 0 {
			usernames[i] = "CN=[DAOTEST]test tester10, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else if i > 0 && i < 10 {
			usernames[i] = "CN=[DAOTEST]test tester0" + strconv.Itoa(i) + ", O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		} else {
			usernames[i] = "CN=[DAOTEST]test tester" + strconv.Itoa(i) + ", O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		}
		user.DistinguishedName = usernames[i]
		user.DisplayName.String = config.GetCommonName(user.DistinguishedName)
		user.DisplayName.Valid = true
		user.CreatedBy = user.DistinguishedName
		_, err = d.CreateUser(&user)
		//createdUser, err = d.CreateUser(&user)
		//log.Printf("User "+strconv.Itoa(i)+" Change Count and Token: %d - %s", createdUser.ChangeCount, createdUser.ChangeToken)
	}

	user.DistinguishedName = "Bob"
	user.CreatedBy = "Bob"
	_, err = d.CreateUser(&user)

	// var user *models.ODUser
	// var user1 models.ODUser
	// user1.DistinguishedName = "CN=[DAOTEST]test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	// user1.DisplayName.String = config.GetCommonName(user1.DistinguishedName)
	// user1.DisplayName.Valid = true
	// user1.CreatedBy = user1.DistinguishedName
	// user, err = d.CreateUser(&user1)
	// log.Printf("User 1 Change Count and Token: %d - %s", user.ChangeCount, user.ChangeToken)
	// var user2 models.ODUser
	// user2.DistinguishedName = "CN=[DAOTEST]test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	// user2.DisplayName.String = config.GetCommonName(user2.DistinguishedName)
	// user2.DisplayName.Valid = true
	// user2.CreatedBy = user2.DistinguishedName
	// user, err = d.CreateUser(&user2)
	// log.Printf("User 2 Change Count and Token: %d - %s", user.ChangeCount, user.ChangeToken)
	// var user10 models.ODUser
	// user10.DistinguishedName = "CN=[DAOTEST]test tester10, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	// user10.DisplayName.String = config.GetCommonName(user10.DistinguishedName)
	// user10.DisplayName.Valid = true
	// user10.CreatedBy = user10.DistinguishedName
	// user, err = d.CreateUser(&user10)
	// log.Printf("User 10 Change Count and Token: %d - %s", user.ChangeCount, user.ChangeToken)

}

func TestTransactionalUpdate(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	tx := db.MustBegin()

	// Add
	typeName := "New Type " + strconv.Itoa(time.Now().UTC().Nanosecond())
	addObjectTypeStatement, err := tx.Preparex(
		`insert object_type set createdBy = ?, name = ?, description = ?, contentConnector = ?`)
	if err != nil {
		log.Printf("Error %v", err)
	}
	_, err = addObjectTypeStatement.Exec("Bob", typeName, "No Decription", "No Content Connector")
	if err != nil {
		log.Printf("Error %v", err)
	}

	// Select 1st time
	dbObjectType1 := models.ODObjectType{}
	if testing.Verbose() {
		jsonData, err := json.MarshalIndent(dbObjectType1, "", "  ")
		if err != nil {
			log.Printf("Error %v", err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}

	getObjectTypeStatement1 := "select * from object_type where name = ?"
	err = tx.Get(&dbObjectType1, getObjectTypeStatement1, typeName)
	if err != nil {
		log.Printf("Error %v", err)
	}
	log.Printf("Change Count = %d", dbObjectType1.ChangeCount)
	if testing.Verbose() {
		jsonData, err := json.MarshalIndent(dbObjectType1, "", "  ")
		if err != nil {
			log.Printf("Error %v", err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}

	// Update (Triggers will alter the changeCount and modifiedDate and changeToken)
	newTypeName := "New Type " + strconv.Itoa(time.Now().UTC().Nanosecond())
	updateObjectTypeStatement, err := tx.Preparex(
		`update object_type set modifiedBy = ?, name = ? where name = ?`)
	if err != nil {
		log.Printf("Error %v", err)
	}
	_, err = updateObjectTypeStatement.Exec("Bob", newTypeName, typeName)
	if err != nil {
		log.Printf("Error %v", err)
	}

	// Select 2nd time
	var dbObjectType2 models.ODObjectType
	getObjectTypeStatement2 := "select * from object_type where name = ?"
	err = tx.Get(&dbObjectType2, getObjectTypeStatement2, newTypeName)
	if err != nil {
		log.Printf("Error %v", err)
	}
	log.Printf("Change Count = %d", dbObjectType2.ChangeCount)
	if testing.Verbose() {
		jsonData, err := json.MarshalIndent(dbObjectType2, "", "  ")
		if err != nil {
			log.Printf("Error %v", err)
		}
		jsonified := string(jsonData)
		fmt.Println(jsonified)
	}
	tx.Commit()
}

package dao_test

import (
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestGetRootObjectsByOwner(t *testing.T) {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		t.Error("Unable to get handle to database: ", err.Error())
	}
	defer db.Close()

	user1 := "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	user2 := "CN=test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"

	// Get root Objects
	resultset, err := dao.GetRootObjectsByOwner(db, "", 1, 1, user1)
	if err != nil {
		t.Error(err)
	}
	// capture how many objects are rooted before changes
	originalTotalRows1 := resultset.TotalRows
	// The same for user2
	resultset, err = dao.GetRootObjectsByOwner(db, "", 1, 1, user2)
	if err != nil {
		t.Failed()
	}
	originalTotalRows2 := resultset.TotalRows

	// Create an object with no parent under user1
	var object1 models.ODObject
	object1.Name = "Test GetRootObjectsByOwner for user1"
	object1.CreatedBy = user1
	object1.TypeName.String = "Test Type"
	object1.TypeName.Valid = true
	err = dao.CreateObject(db, &object1, nil)
	if err != nil {
		t.Error(err)
	}
	if object1.ID == nil {
		t.Error("expected ID to be set")
	}
	if object1.ModifiedBy != object1.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if object1.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Create an object with no parent under user2
	var object2 models.ODObject
	object2.Name = "Test GetRootObjectsByOwner for user2"
	object2.CreatedBy = user2
	object2.TypeName.String = "Test Type"
	object2.TypeName.Valid = true
	err = dao.CreateObject(db, &object2, nil)
	if err != nil {
		t.Error(err)
	}
	if object2.ID == nil {
		t.Error("expected ID to be set")
	}
	if object2.ModifiedBy != object2.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if object2.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Get root Objects again
	resultset, err = dao.GetRootObjectsByOwner(db, "", 1, 1, user1)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != (originalTotalRows1 + 1) {
		t.Error("expected an increase in objects at root")
	}
	resultset, err = dao.GetRootObjectsByOwner(db, "", 1, 1, user2)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != (originalTotalRows2 + 1) {
		t.Error("expected an increase in objects at root")
	}

	// Delete the objects
	err = dao.DeleteObject(db, &object1, true)
	if err != nil {
		t.Error(err)
	}
	err = dao.DeleteObject(db, &object2, true)
	if err != nil {
		t.Error(err)
	}

	// Get root Objects again
	resultset, err = dao.GetRootObjectsByOwner(db, "", 1, 1, user1)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows1 {
		t.Error("expected same number of objects as before the test")
	}
	resultset, err = dao.GetRootObjectsByOwner(db, "", 1, 1, user2)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows2 {
		t.Error("expected same number of objects as before the test")
	}

	db.Close()
}

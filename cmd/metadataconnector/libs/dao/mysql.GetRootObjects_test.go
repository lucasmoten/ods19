package dao_test

import (
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestGetRootObjects(t *testing.T) {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		t.Error("Unable to get handle to database: ", err.Error())
	}
	defer db.Close()

	// Get root Objects
	resultset, err := dao.GetRootObjects(db, "", 1, 1)
	if err != nil {
		t.Failed()
	}
	// capture how many objects are rooted before changes
	originalTotalRows := resultset.TotalRows

	// Create an object with no parent
	var object1 models.ODObject
	object1.Name = "Test GetRootObjects"
	object1.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
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

	// Get root Objects
	resultset, err = dao.GetRootObjects(db, "", 1, 1)
	if err != nil {
		t.Failed()
	}
	if resultset.TotalRows < (originalTotalRows + 1) {
		t.Error("expected an increase in objects at root")
	}

	// Delete the object
	err = dao.DeleteObject(db, &object1, true)
	if err != nil {
		t.Error(err)
	}

	// Get root Objects
	resultset, err = dao.GetRootObjects(db, "", 1, 1)
	if err != nil {
		t.Failed()
	}
	if resultset.TotalRows != originalTotalRows {
		t.Error("expected same number of objects as before the test")
	}

}

package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func TestGetObject(t *testing.T) {
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		t.Error("Unable to get handle to database: ", err.Error())
	}
	defer db.Close()

	var obj models.ODObject
	obj.Name = "Test GetObject"
	obj.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	obj.TypeName.String = "File"
	obj.TypeName.Valid = true
	var acm models.ODACM
	acm.CreatedBy = obj.CreatedBy
	acm.Classification.String = "UNCLASSIFIED"
	acm.Classification.Valid = true
	dao.CreateObject(db, &obj, &acm)
	if obj.ID == nil {
		t.Error("expected ID to be set")
	}
	if obj.ModifiedBy != obj.CreatedBy {
		t.Error("expcted ModifiedBy to match CreatedBy")
	}
	if obj.TypeID == nil {
		t.Error("expected TypeID to be set")
	}
	err = dao.AddPropertyToObject(db, obj.CreatedBy, obj.ID, "Test Property", "Test Property Value", "UNCLASSIFIED")
	if err != nil {
		t.Error(err)
	}

	obj1, err := dao.GetObject(db, &obj, false)
	if err != nil {
		t.Error(err)
	}
	if obj1.Name != "Test GetObject" {
		t.Error("Expected object name to be Test GetObject")
	}
	if len(obj1.Properties) > 0 {
		t.Error("Did not expect properties to be loaded")
	}
	obj2, err := dao.GetObject(db, &obj, true)
	if err != nil {
		t.Error(err)
	}
	if obj2.Name != "Test GetObject" {
		t.Error("Expected object name to be Test GetObject")
	}
	if len(obj2.Properties) != 1 {
		fmt.Println("len(obj2.Properties) = ", len(obj2.Properties))
		t.Error("Expected retrieved object to have 1 property")
	}

	// cleanup
	for _, property := range obj2.Properties {
		dao.DeleteObjectProperty(db, &property)
		if err != nil {
			t.Error(err)
		}
	}
	err = dao.DeleteObject(db, &obj, true)
	if err != nil {
		t.Error(err)
	}
}

package dao_test

import (
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

	// create object
	var obj models.ODObject
	obj.Name = "Test Object for GetObject"
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
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if obj.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// add property
	err = dao.AddPropertyToObject(db, obj.CreatedBy, &obj, "Test Property", "Test Property Value", "UNCLASSIFIED")
	if err != nil {
		t.Error(err)
	}

	// get object with properties
	objectWithProperty, err := dao.GetObject(db, &obj, true)
	if err != nil {
		t.Error(err)
	}
	if len(objectWithProperty.Properties) != 1 {
		t.Error("Expected one property on the object")
	} else {
		if objectWithProperty.Properties[0].Name != "Test Property" {
			t.Error("Expected property name to be Test Property")
		}
		if objectWithProperty.Properties[0].Value.String != "Test Property Value" {
			t.Error("Expected property value to be Test Property Value")
		}
	}

	// get object without properties
	objectWithoutProperty, err := dao.GetObject(db, &obj, false)
	if err != nil {
		t.Error(err)
	}
	if len(objectWithoutProperty.Properties) != 0 {
		t.Error("Expected zero properties on the object")
	}

	// delete the Property
	if len(objectWithProperty.Properties) > 0 {
		theProperty := objectWithProperty.Properties[0]
		err = dao.DeleteObjectProperty(db, &theProperty)
		if err != nil {
			t.Error(err)
		}
	}

	// delete the object
	err = dao.DeleteObject(db, objectWithProperty, true)
	if err != nil {
		t.Error(err)
	}

	db.Close()
}

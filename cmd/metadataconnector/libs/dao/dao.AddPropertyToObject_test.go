package dao_test

import (
	"log"
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOAddPropertyToObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if db == nil {
		log.Fatal("db is nil")
	}

	// create object
	var obj models.ODObject
	obj.Name = "Test Object for Adding Property"
	obj.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	obj.TypeName.String = "File"
	obj.TypeName.Valid = true
	obj.RawAcm.String = testhelpers.ValidACMUnclassified
	dbObject, err := d.CreateObject(&obj)
	if dbObject.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbObject.ModifiedBy != obj.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbObject.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// add property
	var property models.ODProperty
	property.CreatedBy = obj.CreatedBy
	property.Name = "Test Property"
	property.Value.String = "Test Property Value"
	property.Value.Valid = true
	property.ClassificationPM.String = "UNCLASSIFIED"
	property.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbObject, &property)
	if err != nil {
		t.Error(err)
	}

	// get object with properties
	objectWithProperty, err := d.GetObject(dbObject, true)
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

		// delete the Property
		theProperty := objectWithProperty.Properties[0]
		err = d.DeleteObjectProperty(theProperty)
		if err != nil {
			t.Error(err)
		}
	}

	// delete the object
	err = d.DeleteObject(objectWithProperty, true)
	if err != nil {
		t.Error(err)
	}
}

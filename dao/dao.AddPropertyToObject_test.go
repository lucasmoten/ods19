package dao_test

import (
	"log"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
)

func TestDAOAddPropertyToObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if db == nil {
		log.Fatal("db is nil")
	}

	user := models.ODUser{DistinguishedName: usernames[1]}

	// create object
	var obj models.ODObject
	obj.Name = "Test Object for Adding Property"
	obj.CreatedBy = usernames[1]
	obj.TypeName = models.ToNullString("File")
	obj.RawAcm.String = ValidACMUnclassified
	objectType, err := d.GetObjectTypeByName(obj.TypeName.String, true, obj.CreatedBy)
	if err != nil {
		t.Error(err)
	} else {
		obj.TypeID = objectType.ID
	}
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
	property.Value = models.ToNullString("Test Property Value")
	property.ClassificationPM = models.ToNullString("UNCLASSIFIED")
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
	err = d.DeleteObject(user, objectWithProperty, true)
	if err != nil {
		t.Error(err)
	}
}

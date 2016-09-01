package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetObject(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	var obj models.ODObject
	obj.Name = "Test Object for GetObject"
	obj.CreatedBy = usernames[1]
	obj.TypeName.String = "File"
	obj.TypeName.Valid = true
	permissions := make([]models.ODObjectPermission, 1)
	permissions[0].Grantee = obj.CreatedBy
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions[0].Grantee)
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.String = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.Valid = true
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	obj.Permissions = permissions
	properties := make([]models.ODObjectPropertyEx, 1)
	properties[0].Name = "Test Property in TestDAOGetObject"
	properties[0].Value.String = "Test Property Value"
	properties[0].Value.Valid = true
	properties[0].ClassificationPM.String = "UNCLASSIFIED"
	properties[0].ClassificationPM.Valid = true
	obj.Properties = properties
	obj.RawAcm.String = testhelpers.ValidACMUnclassified
	dbObject, err := d.CreateObject(&obj)
	if err != nil {
		t.Error(err)
	}
	if dbObject.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbObject.ModifiedBy != obj.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbObject.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// // add property
	// var property models.ODProperty
	// property.CreatedBy = dbObject.CreatedBy
	// property.Name = "Test Property"
	// property.Value.String = "Test Property Value"
	// property.Value.Valid = true
	// property.ClassificationPM.String = "UNCLASSIFIED"
	// property.ClassificationPM.Valid = true
	// _, err = d.AddPropertyToObject(dbObject, &property)
	// if err != nil {
	// 	t.Error(err)
	// }

	// get object with properties
	objectWithProperty, err := d.GetObject(dbObject, true)
	if err != nil {
		t.Error(err)
	}
	if len(objectWithProperty.Properties) != 1 {

		// jsonData, _ := json.MarshalIndent(objectWithProperty, "", "  ")
		// jsonified := string(jsonData)
		// fmt.Println(jsonified)

		t.Errorf("Expected one property on the object, got %d", len(objectWithProperty.Properties))
	} else {
		if objectWithProperty.Properties[0].Name != "Test Property in TestDAOGetObject" {
			t.Error("Expected property name to be Test Property in TestDAOGetObject")
		}
		if objectWithProperty.Properties[0].Value.String != "Test Property Value" {
			t.Error("Expected property value to be Test Property Value")
		}
	}

	// get object without properties
	objectWithoutProperty, err := d.GetObject(dbObject, false)
	if err != nil {
		t.Error(err)
	}
	if len(objectWithoutProperty.Properties) != 0 {
		t.Error("Expected zero properties on the object")
	}

	// delete the Property
	if len(objectWithProperty.Properties) > 0 {
		theProperty := objectWithProperty.Properties[0]
		err = d.DeleteObjectProperty(theProperty)
		if err != nil {
			t.Error(err)
		}
	}

	// delete the object
	user := models.ODUser{DistinguishedName: objectWithProperty.CreatedBy}
	err = d.DeleteObject(user, objectWithProperty, true)
	if err != nil {
		t.Error(err)
	}
}

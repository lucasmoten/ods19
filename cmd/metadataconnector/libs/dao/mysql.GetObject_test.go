package dao_test

//
// import (
// 	"encoding/json"
// 	"fmt"
// 	"testing"
//
// 	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
// 	"decipher.com/oduploader/metadata/models"
// )
//
// func TTestGetObject(t *testing.T) {
// 	// create object
// 	var obj models.ODObject
// 	obj.Name = "Test Object for GetObject"
// 	obj.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
// 	obj.TypeName.String = "File"
// 	obj.TypeName.Valid = true
// 	var acm models.ODACM
// 	acm.CreatedBy = obj.CreatedBy
// 	acm.Classification.String = "UNCLASSIFIED"
// 	acm.Classification.Valid = true
// 	dao.CreateObject(db, &obj, &acm)
// 	if obj.ID == nil {
// 		t.Error("expected ID to be set")
// 	}
// 	if obj.ModifiedBy != obj.CreatedBy {
// 		t.Error("expected ModifiedBy to match CreatedBy")
// 	}
// 	if obj.TypeID == nil {
// 		t.Error("expected TypeID to be set")
// 	}
//
// 	// add property
// 	var property models.ODProperty
// 	property.Name = "Test Property"
// 	property.Value.String = "Test Property Value"
// 	property.Value.Valid = true
// 	property.ClassificationPM.String = "UNCLASSIFIED"
// 	property.ClassificationPM.Valid = true
// 	err := dao.AddPropertyToObject(db, obj.CreatedBy, &obj, &property)
// 	if err != nil {
// 		t.Error(err)
// 	}
//
// 	// get object with properties
// 	objectWithProperty, err := dao.GetObject(db, &obj, true)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if len(objectWithProperty.Properties) != 1 {
//
// 		jsonData, _ := json.MarshalIndent(objectWithProperty, "", "  ")
// 		jsonified := string(jsonData)
// 		fmt.Println(jsonified)
//
// 		t.Errorf("Expected one property on the object, got %d", len(objectWithProperty.Properties))
// 	} else {
// 		if objectWithProperty.Properties[0].Name != "Test Property" {
// 			t.Error("Expected property name to be Test Property")
// 		}
// 		if objectWithProperty.Properties[0].Value.String != "Test Property Value" {
// 			t.Error("Expected property value to be Test Property Value")
// 		}
// 	}
//
// 	// get object without properties
// 	objectWithoutProperty, err := dao.GetObject(db, &obj, false)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if len(objectWithoutProperty.Properties) != 0 {
// 		t.Error("Expected zero properties on the object")
// 	}
//
// 	// delete the Property
// 	if len(objectWithProperty.Properties) > 0 {
// 		theProperty := objectWithProperty.Properties[0]
// 		err = dao.DeleteObjectProperty(db, &theProperty)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 	}
//
// 	// delete the object
// 	err = dao.DeleteObject(db, objectWithProperty, true)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

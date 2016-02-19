package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/oduploader/metadata/models"
)

func TestDAOGetChildObjectsWithProperties(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	// create parent object
	var parent models.ODObject
	parent.Name = "Test Parent Object for GetChildObjectsWithProperties"
	parent.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	parent.TypeName.String = "File"
	parent.TypeName.Valid = true
	var acm models.ODACM
	acm.CreatedBy = parent.CreatedBy
	acm.Classification.String = "UNCLASSIFIED"
	acm.Classification.Valid = true
	d.CreateObject(&parent, &acm)
	if parent.ID == nil {
		t.Error("expected ID to be set")
	}
	if parent.ModifiedBy != parent.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if parent.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// create child 1
	var child1 models.ODObject
	child1.Name = "Test Child Object 1 for GetChildObjectsWithProperties"
	child1.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	child1.TypeName.String = "File"
	child1.TypeName.Valid = true
	child1.ParentID = parent.ID
	var acm1 models.ODACM
	acm1.CreatedBy = child1.CreatedBy
	acm1.Classification.String = "UNCLASSIFIED"
	acm1.Classification.Valid = true
	d.CreateObject(&child1, &acm1)
	if child1.ID == nil {
		t.Error("expected ID to be set")
	}
	if child1.ModifiedBy != child1.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if child1.TypeID == nil {
		t.Error("expected TypeID to be set")
	}
	// property on child 1
	var property1 models.ODProperty
	property1.Name = "Test Property C1P1"
	property1.Value.String = "Test Property 1 Value"
	property1.Value.Valid = true
	property1.ClassificationPM.String = "UNCLASSIFIED"
	property1.ClassificationPM.Valid = true
	err := d.AddPropertyToObject(child1.CreatedBy, &child1, &property1)
	if err != nil {
		t.Error(err)
	}
	var property2 models.ODProperty
	property2.Name = "Test Property C1P2"
	property2.Value.String = "Test Property 2 Value"
	property2.Value.Valid = true
	property2.ClassificationPM.String = "UNCLASSIFIED"
	property2.ClassificationPM.Valid = true
	err = d.AddPropertyToObject(child1.CreatedBy, &child1, &property2)
	if err != nil {
		t.Error(err)
	}

	// create child 2
	var child2 models.ODObject
	child2.Name = "Test Child Object 2 for GetChildObjectsWithProperties"
	child2.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	child2.TypeName.String = "File"
	child2.TypeName.Valid = true
	child2.ParentID = parent.ID
	var acm2 models.ODACM
	acm2.CreatedBy = child2.CreatedBy
	acm2.Classification.String = "UNCLASSIFIED"
	acm2.Classification.Valid = true
	d.CreateObject(&child2, &acm2)
	if child2.ID == nil {
		t.Error("expected ID to be set")
	}
	if child2.ModifiedBy != child2.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if child2.TypeID == nil {
		t.Error("expected TypeID to be set")
	}
	// property on child 1
	var propertyc2p1 models.ODProperty
	propertyc2p1.Name = "Test Property C2P1"
	propertyc2p1.Value.String = "Test Property 1 Value"
	propertyc2p1.Value.Valid = true
	propertyc2p1.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p1.ClassificationPM.Valid = true
	err = d.AddPropertyToObject(child2.CreatedBy, &child2, &propertyc2p1)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p2 models.ODProperty
	propertyc2p2.Name = "Test Property C2P2"
	propertyc2p2.Value.String = "Test Property 2 Value"
	propertyc2p2.Value.Valid = true
	propertyc2p2.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p2.ClassificationPM.Valid = true
	err = d.AddPropertyToObject(child2.CreatedBy, &child2, &propertyc2p2)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p3 models.ODProperty
	propertyc2p3.Name = "Test Property C2P3"
	propertyc2p3.Value.String = "Test Property 3 Value"
	propertyc2p3.Value.Valid = true
	propertyc2p3.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p3.ClassificationPM.Valid = true
	err = d.AddPropertyToObject(child2.CreatedBy, &child2, &propertyc2p3)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p4 models.ODProperty
	propertyc2p4.Name = "Test Property C2P4"
	propertyc2p4.Value.String = "Test Property 4 Value"
	propertyc2p4.Value.Valid = true
	propertyc2p4.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p4.ClassificationPM.Valid = true
	err = d.AddPropertyToObject(child2.CreatedBy, &child2, &propertyc2p4)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p5 models.ODProperty
	propertyc2p5.Name = "Test Property C2P5"
	propertyc2p5.Value.String = "Test Property 5 Value"
	propertyc2p5.Value.Valid = true
	propertyc2p5.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p5.ClassificationPM.Valid = true
	err = d.AddPropertyToObject(child2.CreatedBy, &child2, &propertyc2p5)
	if err != nil {
		t.Error(err)
	}

	// Get child objects with properties from a single page of up to 10
	resultset, err := d.GetChildObjectsWithProperties("", 1, 10, &parent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 2 {
		t.Error("Expected 2 children")
	}
	if len(resultset.Objects) != 2 {
		t.Error("Expected 2 objects")
	} else {
		if resultset.Objects[0].Name != child1.Name {
			t.Error(fmt.Errorf("Name of first child didn't match expected value. Got %s, expected %s", resultset.Objects[0].Name, child1.Name))
		}
		if resultset.Objects[1].Name != child2.Name {
			t.Error(fmt.Errorf("Name of second child didn't match expected value. Got %s, expected %s", resultset.Objects[1].Name, child2.Name))
		}
		if len(resultset.Objects[0].Properties) != 2 {
			t.Error(fmt.Errorf("Expected first child to have 2 properties, but has %d", len(resultset.Objects[0].Properties)))
		}
		if len(resultset.Objects[1].Properties) != 5 {
			t.Error(fmt.Errorf("Expected second child to have 5 properties, but has %d", len(resultset.Objects[1].Properties)))
		}
	}

	// Get from first page of 1, then second page of 1
	resultset, err = d.GetChildObjectsWithProperties("", 1, 1, &parent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 2 {
		t.Error("Expected 2 children")
	}
	if len(resultset.Objects) != 1 {
		t.Error("Expected 1 objects")
	} else {
		if resultset.Objects[0].Name != child1.Name {
			t.Error("Name of first child didn't match expected value")
		}
		if len(resultset.Objects[0].Properties) != 2 {
			t.Error("Expected first child to have 2 properties")
		}
	}
	resultset, err = d.GetChildObjectsWithProperties("", 2, 1, &parent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 2 {
		t.Error("Expected 2 children")
	}
	if len(resultset.Objects) != 1 {
		t.Error("Expected 1 objects")
	} else {
		if resultset.Objects[0].Name != child2.Name {
			t.Error("Name of first child didn't match expected value")
		}
		if len(resultset.Objects[0].Properties) != 5 {
			t.Error("Expected child on page 2 to have 5 properties")
		}
	}

	// cleanup
	for _, object := range resultset.Objects {
		for _, property := range object.Properties {
			d.DeleteObjectProperty(&property)
		}
		d.DeleteObject(&object, true)
	}
	d.DeleteObject(&parent, true)

}
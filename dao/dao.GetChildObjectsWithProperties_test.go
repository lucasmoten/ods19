package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetChildObjectsWithProperties(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	// create parent object
	var parent models.ODObject
	parent.Name = "Test Parent Object for GetChildObjectsWithProperties"
	parent.CreatedBy = usernames[1]
	parent.TypeName.String = "File"
	parent.TypeName.Valid = true
	parent.RawAcm.String = testhelpers.ValidACMUnclassified
	dbParent, err := d.CreateObject(&parent)
	if dbParent.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbParent.ModifiedBy != parent.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbParent.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// create child 1
	var child1 models.ODObject
	child1.Name = "Test Child Object 1 for GetChildObjectsWithProperties"
	child1.CreatedBy = usernames[1]
	child1.TypeName.String = "File"
	child1.TypeName.Valid = true
	child1.ParentID = dbParent.ID
	child1.RawAcm.String = testhelpers.ValidACMUnclassified
	dbChild1, err := d.CreateObject(&child1)
	if dbChild1.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbChild1.ModifiedBy != child1.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbChild1.TypeID == nil {
		t.Error("expected TypeID to be set")
	}
	// property on child 1
	var property1 models.ODProperty
	property1.CreatedBy = child1.CreatedBy
	property1.Name = "Test Property C1P1"
	property1.Value.String = "Test Property 1 Value"
	property1.Value.Valid = true
	property1.ClassificationPM.String = "UNCLASSIFIED"
	property1.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbChild1, &property1)
	if err != nil {
		t.Error(err)
	}
	var property2 models.ODProperty
	property2.CreatedBy = child1.CreatedBy
	property2.Name = "Test Property C1P2"
	property2.Value.String = "Test Property 2 Value"
	property2.Value.Valid = true
	property2.ClassificationPM.String = "UNCLASSIFIED"
	property2.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbChild1, &property2)
	if err != nil {
		t.Error(err)
	}

	// create child 2
	var child2 models.ODObject
	child2.Name = "Test Child Object 2 for GetChildObjectsWithProperties"
	child2.CreatedBy = usernames[1]
	child2.TypeName.String = "File"
	child2.TypeName.Valid = true
	child2.ParentID = dbParent.ID
	child2.RawAcm.String = testhelpers.ValidACMUnclassified
	dbChild2, err := d.CreateObject(&child2)
	if dbChild2.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbChild2.ModifiedBy != child2.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbChild2.TypeID == nil {
		t.Error("expected TypeID to be set")
	}
	// property on child 1
	var propertyc2p1 models.ODProperty
	propertyc2p1.CreatedBy = child2.CreatedBy
	propertyc2p1.Name = "Test Property C2P1"
	propertyc2p1.Value.String = "Test Property 1 Value"
	propertyc2p1.Value.Valid = true
	propertyc2p1.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p1.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p1)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p2 models.ODProperty
	propertyc2p2.CreatedBy = child2.CreatedBy
	propertyc2p2.Name = "Test Property C2P2"
	propertyc2p2.Value.String = "Test Property 2 Value"
	propertyc2p2.Value.Valid = true
	propertyc2p2.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p2.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p2)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p3 models.ODProperty
	propertyc2p3.CreatedBy = child2.CreatedBy
	propertyc2p3.Name = "Test Property C2P3"
	propertyc2p3.Value.String = "Test Property 3 Value"
	propertyc2p3.Value.Valid = true
	propertyc2p3.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p3.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p3)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p4 models.ODProperty
	propertyc2p4.CreatedBy = child2.CreatedBy
	propertyc2p4.Name = "Test Property C2P4"
	propertyc2p4.Value.String = "Test Property 4 Value"
	propertyc2p4.Value.Valid = true
	propertyc2p4.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p4.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p4)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p5 models.ODProperty
	propertyc2p5.CreatedBy = child2.CreatedBy
	propertyc2p5.Name = "Test Property C2P5"
	propertyc2p5.Value.String = "Test Property 5 Value"
	propertyc2p5.Value.Valid = true
	propertyc2p5.ClassificationPM.String = "UNCLASSIFIED"
	propertyc2p5.ClassificationPM.Valid = true
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p5)
	if err != nil {
		t.Error(err)
	}

	// Get child objects with properties from a single page of up to 10
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 10, SortSettings: []dao.SortSetting{dao.SortSetting{SortField: "createddate", SortAscending: true}}}
	resultset, err := d.GetChildObjectsWithProperties(pagingRequest, dbParent)
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
	pagingRequest.PageSize = 1
	resultset, err = d.GetChildObjectsWithProperties(pagingRequest, dbParent)
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
	pagingRequest.PageNumber = 2
	resultset, err = d.GetChildObjectsWithProperties(pagingRequest, dbParent)
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
	user := models.ODUser{DistinguishedName: dbParent.CreatedBy}
	for _, object := range resultset.Objects {
		for _, property := range object.Properties {
			d.DeleteObjectProperty(property)
		}
		d.DeleteObject(user, object, true)
	}
	d.DeleteObject(user, dbParent, true)

}

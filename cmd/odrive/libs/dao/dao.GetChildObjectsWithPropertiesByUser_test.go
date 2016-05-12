package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetChildObjectsWithPropertiesByUser(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	user1 := usernames[1]
	user2 := usernames[2]

	// create parent object
	var parent models.ODObject
	parent.Name = "Test Parent Object for GetChildObjectsWithPropertiesByUser"
	parent.CreatedBy = user1
	parent.TypeName.String = "File"
	parent.TypeName.Valid = true
	parent.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions := make([]models.ODObjectPermission, 1)
	permissions[0].CreatedBy = user1
	permissions[0].Grantee = user1
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	parent.Permissions = permissions
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
	child1.Name = "Test Child Object 1 for GetChildObjectsWithPropertiesByUser"
	child1.CreatedBy = user1
	child1.TypeName.String = "File"
	child1.TypeName.Valid = true
	child1.ParentID = dbParent.ID
	child1.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions1 := make([]models.ODObjectPermission, 1)
	permissions1[0].CreatedBy = user1
	permissions1[0].Grantee = user1
	permissions1[0].AllowCreate = true
	permissions1[0].AllowRead = true
	permissions1[0].AllowUpdate = true
	permissions1[0].AllowDelete = true
	child1.Permissions = permissions1
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
	property1.CreatedBy = dbChild1.CreatedBy
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
	property2.CreatedBy = dbChild1.CreatedBy
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
	child2.Name = "Test Child Object 2 for GetChildObjectsWithPropertiesByUser"
	child2.CreatedBy = user1
	child2.TypeName.String = "File"
	child2.TypeName.Valid = true
	child2.ParentID = dbParent.ID
	child2.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions2 := make([]models.ODObjectPermission, 1)
	permissions2[0].CreatedBy = user1
	permissions2[0].Grantee = user1
	permissions2[0].AllowCreate = true
	permissions2[0].AllowRead = true
	permissions2[0].AllowUpdate = true
	permissions2[0].AllowDelete = true
	child2.Permissions = permissions2
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
	propertyc2p1.CreatedBy = dbChild2.CreatedBy
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
	propertyc2p2.CreatedBy = dbChild2.CreatedBy
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
	propertyc2p3.CreatedBy = dbChild2.CreatedBy
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
	propertyc2p4.CreatedBy = dbChild2.CreatedBy
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
	propertyc2p5.CreatedBy = dbChild2.CreatedBy
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
	user := models.ODUser{DistinguishedName: user1}
	pagingRequest := protocol.PagingRequest{PageNumber: 1, PageSize: 10, SortSettings: []protocol.SortSetting{protocol.SortSetting{SortField: "name", SortAscending: true}}}
	resultset, err := d.GetChildObjectsWithPropertiesByUser(user, pagingRequest, dbParent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 2 {
		t.Error(fmt.Errorf("Resultset had %d totalrows", resultset.TotalRows))
		t.Error("Expected 2 children")
	}
	if len(resultset.Objects) != 2 {
		t.Error("Expected 2 objects")
	} else {
		if resultset.Objects[0].Name != child1.Name {
			t.Error(fmt.Errorf("Name of first child '%s' didn't match expected value: %s", resultset.Objects[0].Name, child1.Name))
		}
		if resultset.Objects[1].Name != child2.Name {
			t.Error(fmt.Errorf("Name of second child '%s' didn't match expected value: %s", resultset.Objects[1].Name, child2.Name))
		}
		if len(resultset.Objects[0].Properties) != 2 {
			t.Error(fmt.Errorf("Expected first child to have 2 properties, but it had %d", len(resultset.Objects[0].Properties)))
		}
		if len(resultset.Objects[1].Properties) != 5 {
			t.Error(fmt.Errorf("Expected second child to have 5 properties, but it had %d", len(resultset.Objects[1].Properties)))
		}
	}

	// Get from first page of 1, then second page of 1
	pagingRequest.PageSize = 1
	resultset, err = d.GetChildObjectsWithPropertiesByUser(user, pagingRequest, dbParent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 2 {
		t.Error(fmt.Errorf("Resultset had %d totalrows", resultset.TotalRows))
		t.Error("Expected 2 children")
	}
	if len(resultset.Objects) != 1 {
		t.Error("Expected 1 objects")
	} else {
		if resultset.Objects[0].Name != child1.Name {
			t.Error("Name of first child didn't match expected value")
		}
		if len(resultset.Objects[0].Properties) != 2 {
			t.Error(fmt.Errorf("Expected first child to have 2 properties, but it had %d", len(resultset.Objects[0].Properties)))
		}
	}
	pagingRequest.PageNumber = 2
	resultset, err = d.GetChildObjectsWithPropertiesByUser(user, pagingRequest, dbParent)
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
			t.Error(fmt.Errorf("Expected child on page 2 to have 5 properties, but it had %d", len(resultset.Objects[0].Properties)))
		}
	}
	user.DistinguishedName = user2
	pagingRequest.PageNumber = 1
	pagingRequest.PageSize = 10
	resultset, err = d.GetChildObjectsWithPropertiesByUser(user, pagingRequest, parent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 0 {
		t.Error("Expected 0 children for user2")
	}

	// cleanup
	for _, object := range resultset.Objects {
		for _, property := range object.Properties {
			d.DeleteObjectProperty(property)
		}
		d.DeleteObject(object, true)
	}
	d.DeleteObject(dbParent, true)

}

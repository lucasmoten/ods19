package dao_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/server"
	"github.com/deciphernow/object-drive-server/util"
)

func TestDAOGetChildObjectsWithProperties(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	guid, _ := util.NewGUID()
	timeSuffix := strconv.FormatInt(time.Now().Unix(), 10) + guid

	t.Logf("Create parent object")
	var parent models.ODObject
	parent.Name = "Test Parent Object for GetChildObjectsWithProperties" + timeSuffix
	parent.CreatedBy = usernames[1]
	parent.TypeName = models.ToNullString("File")
	parent.RawAcm.String = server.ValidACMUnclassified
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

	t.Logf("Create child 1")
	var child1 models.ODObject
	child1.Name = "Test Child Object 1 for GetChildObjectsWithProperties" + timeSuffix
	child1.CreatedBy = usernames[1]
	child1.TypeName = models.ToNullString("File")
	child1.ParentID = dbParent.ID
	child1.RawAcm.String = server.ValidACMUnclassified
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
	property1.Value = models.ToNullString("Test Property 1 Value")
	property1.ClassificationPM = models.ToNullString("UNCLASSIFIED")
	_, err = d.AddPropertyToObject(dbChild1, &property1)
	if err != nil {
		t.Error(err)
	}
	var property2 models.ODProperty
	property2.CreatedBy = child1.CreatedBy
	property2.Name = "Test Property C1P2"
	property2.Value = models.ToNullString("Test Property 2 Value")
	property2.ClassificationPM = models.ToNullString("UNCLASSIFIED")
	_, err = d.AddPropertyToObject(dbChild1, &property2)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)
	t.Logf("Create child 2")
	var child2 models.ODObject
	child2.Name = "Test Child Object 2 for GetChildObjectsWithProperties" + timeSuffix
	child2.CreatedBy = usernames[1]
	child2.TypeName = models.ToNullString("File")
	child2.ParentID = dbParent.ID
	child2.RawAcm.String = server.ValidACMUnclassified
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
	propertyc2p1.Value = models.ToNullString("Test Property 1 Value")
	propertyc2p1.ClassificationPM = models.ToNullString("UNCLASSIFIED")
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p1)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p2 models.ODProperty
	propertyc2p2.CreatedBy = child2.CreatedBy
	propertyc2p2.Name = "Test Property C2P2"
	propertyc2p2.Value = models.ToNullString("Test Property 2 Value")
	propertyc2p2.ClassificationPM = models.ToNullString("UNCLASSIFIED")
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p2)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p3 models.ODProperty
	propertyc2p3.CreatedBy = child2.CreatedBy
	propertyc2p3.Name = "Test Property C2P3"
	propertyc2p3.Value = models.ToNullString("Test Property 3 Value")
	propertyc2p3.ClassificationPM = models.ToNullString("UNCLASSIFIED")
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p3)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p4 models.ODProperty
	propertyc2p4.CreatedBy = child2.CreatedBy
	propertyc2p4.Name = "Test Property C2P4"
	propertyc2p4.Value = models.ToNullString("Test Property 4 Value")
	propertyc2p4.ClassificationPM = models.ToNullString("UNCLASSIFIED")
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p4)
	if err != nil {
		t.Error(err)
	}
	var propertyc2p5 models.ODProperty
	propertyc2p5.CreatedBy = child2.CreatedBy
	propertyc2p5.Name = "Test Property C2P5"
	propertyc2p5.Value = models.ToNullString("Test Property 5 Value")
	propertyc2p5.ClassificationPM = models.ToNullString("UNCLASSIFIED")
	_, err = d.AddPropertyToObject(dbChild2, &propertyc2p5)
	if err != nil {
		t.Error(err)
	}

	t.Logf("Get child objects with properties from a single page of up to 10")
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

	t.Logf("Get from first page of 1, ...")
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
	t.Logf("...then second page of 1")
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
}

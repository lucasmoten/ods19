package dao_test

import (
	"bytes"
	"strings"
	"testing"

	"decipher.com/oduploader/metadata/models"
)

func TestDAOGetChildObjectsByOwner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var parent models.ODObject
	var child1 models.ODObject
	var child2 models.ODObject

	// Create our parent object
	parent.Name = "Test GetChildObjectsByOwner Parent"
	parent.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	parent.TypeName.String = "Test Type"
	parent.TypeName.Valid = true
	err := d.CreateObject(&parent, nil)
	if err != nil {
		t.Error(err)
	} else {
		if parent.ID == nil {
			t.Error("expected ID to be set")
		}
		if parent.ModifiedBy != parent.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if parent.TypeID == nil {
			t.Error("expected TypeID to be set")
		}

		// Create our child object from TP1
		child1.Name = "Test GetChildObjectsByOwner Child by TP1"
		child1.CreatedBy = "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		child1.ParentID = parent.ID
		child1.TypeName.String = "Test Type"
		child1.TypeName.Valid = true
		err = d.CreateObject(&child1, nil)
		if err != nil {
			t.Error(err)
		}
		if child1.ID == nil {
			t.Error("expected ID to be set")
		}
		if child1.ModifiedBy != child1.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if child1.TypeID == nil {
			t.Error("expected TypeID to be set")
		}
		if !bytes.Equal(child1.ParentID, parent.ID) {
			t.Error("expected child parentID to match parent ID")
		}

		// Create our child object from TP2
		child2.Name = "Test GetChildObjectsByOwner Child by TP2"
		child2.CreatedBy = "CN=test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		child2.ParentID = parent.ID
		child2.TypeName.String = "Test Type"
		child2.TypeName.Valid = true
		err = d.CreateObject(&child2, nil)
		if err != nil {
			t.Error(err)
		}
		if child2.ID == nil {
			t.Error("expected ID to be set")
		}
		if child2.ModifiedBy != child2.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if child2.TypeID == nil {
			t.Error("expected TypeID to be set")
		}
		if !bytes.Equal(child2.ParentID, parent.ID) {
			t.Error("expected child parentID to match parent ID")
		}
		resultset, err := d.GetChildObjectsByOwner("", 1, 10, &parent, child2.CreatedBy)
		if err != nil {
			t.Error(err)
		}
		if resultset.TotalRows != 1 {
			t.Error("expected 1 child")
		}
		if resultset.Objects[0].ModifiedBy != child2.CreatedBy {
			t.Error("expected result modifiedBy to match child2 created by")
		}
		if !strings.Contains(resultset.Objects[0].ModifiedBy, "tester02") {
			t.Error("expected result ModifiedBy to be by tester02")
		}
	}

	// cleanup
	err = d.DeleteObject(&parent, true)
	if err != nil {
		t.Error(err)
	}
}

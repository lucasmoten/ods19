package dao_test

import (
	"bytes"
	"testing"

	"decipher.com/oduploader/metadata/models"
)

func TestDAOGetChildObjects(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	// Create our parent object
	var parent models.ODObject
	parent.Name = "Test GetChildObjects Parent"
	parent.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	parent.TypeName.String = "Test Type"
	parent.TypeName.Valid = true
	err := d.CreateObject(&parent, nil)
	if err != nil {
		t.Error(err)
	}
	if parent.ID == nil {
		t.Error("expected ID to be set")
	}
	if parent.ModifiedBy != parent.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if parent.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Create our child object
	var child models.ODObject
	child.Name = "Test GetChildObjects Child"
	child.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	child.ParentID = parent.ID
	child.TypeName.String = "Test Type"
	child.TypeName.Valid = true
	err = d.CreateObject(&child, nil)
	if err != nil {
		t.Error(err)
	}
	if child.ID == nil {
		t.Error("expected ID to be set")
	}
	if child.ModifiedBy != child.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if child.TypeID == nil {
		t.Error("expected TypeID to be set")
	}
	if !bytes.Equal(child.ParentID, parent.ID) {
		t.Error("expected child parentID to match parent ID")
	}

	resultset, err := d.GetChildObjects("", 1, 10, &parent)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != 1 {
		t.Error("expected 1 child")
	}

	// cleanup
	err = d.DeleteObject(&parent, true)
	if err != nil {
		t.Error(err)
	}

}

package dao_test

import (
	"testing"

	"decipher.com/oduploader/metadata/models"
)

func TestDAOGetRootObjects(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// Get root Objects
	resultset, err := d.GetRootObjects("", 1, 1)
	if err != nil {
		t.Error(err)
	}
	// capture how many objects are rooted before changes
	originalTotalRows := resultset.TotalRows

	// Create an object with no parent
	var object1 models.ODObject
	object1.Name = "Test GetRootObjects"
	object1.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	object1.TypeName.String = "Test Type"
	object1.TypeName.Valid = true
	err = d.CreateObject(&object1, nil)
	if err != nil {
		t.Error(err)
	}
	if object1.ID == nil {
		t.Error("expected ID to be set")
	}
	if object1.ModifiedBy != object1.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if object1.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Get root Objects
	resultset, err = d.GetRootObjects("", 1, 1)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows < (originalTotalRows + 1) {
		t.Error("expected an increase in objects at root")
	}

	// Delete the object
	err = d.DeleteObject(&object1, true)
	if err != nil {
		t.Error(err)
	}

	// Get root Objects
	resultset, err = d.GetRootObjects("", 1, 1)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows {
		t.Error("expected same number of objects as before the test")
	}
}

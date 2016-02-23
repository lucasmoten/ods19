package dao_test

import (
	"testing"

	"decipher.com/oduploader/metadata/models"
)

func TestDAOGetRootObjectsByUser(t *testing.T) {
	user1 := usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	user2 := usernames[2] // "CN=test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"

	// Get root Objects
	resultset, err := d.GetRootObjectsByUser("", 1, 1, user1)
	if err != nil {
		t.Error(err)
	}
	// capture how many objects are rooted before changes
	originalTotalRows1 := resultset.TotalRows
	// The same for user2
	resultset, err = d.GetRootObjectsByUser("", 1, 1, user2)
	if err != nil {
		t.Failed()
	}
	originalTotalRows2 := resultset.TotalRows

	// Create an object with no parent under user1
	var object1 models.ODObject
	object1.Name = "Test GetRootObjectsByUser for user1"
	object1.CreatedBy = user1
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

	// Create an object with no parent under user2
	var object2 models.ODObject
	object2.Name = "Test GetRootObjectsByUser for user2"
	object2.CreatedBy = user2
	object2.TypeName.String = "Test Type"
	object2.TypeName.Valid = true
	err = d.CreateObject(&object2, nil)
	if err != nil {
		t.Error(err)
	}
	if object2.ID == nil {
		t.Error("expected ID to be set")
	}
	if object2.ModifiedBy != object2.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if object2.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Get root Objects again
	resultset, err = d.GetRootObjectsByUser("", 1, 1, user1)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != (originalTotalRows1 + 1) {
		t.Error("expected an increase in objects at root")
	}
	resultset, err = d.GetRootObjectsByUser("", 1, 1, user2)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != (originalTotalRows2 + 1) {
		t.Error("expected an increase in objects at root")
	}

	// Delete the objects
	err = d.DeleteObject(&object1, true)
	if err != nil {
		t.Error(err)
	}
	err = d.DeleteObject(&object2, true)
	if err != nil {
		t.Error(err)
	}

	// Get root Objects again
	resultset, err = d.GetRootObjectsByUser("", 1, 1, user1)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows1 {
		t.Error("expected same number of objects as before the test")
	}
	resultset, err = d.GetRootObjectsByUser("", 1, 1, user2)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows2 {
		t.Error("expected same number of objects as before the test")
	}
}

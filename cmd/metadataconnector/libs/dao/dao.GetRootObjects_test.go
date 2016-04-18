package dao_test

import (
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetRootObjects(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	pagingRequest := protocol.PagingRequest{PageNumber: 1, PageSize: 1}
	// Get root Objects
	resultset, err := d.GetRootObjects(pagingRequest)
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
	object1.RawAcm.String = testhelpers.ValidACMUnclassified
	dbObject1, err := d.CreateObject(&object1)
	if err != nil {
		t.Error(err)
	}
	if dbObject1.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbObject1.ModifiedBy != object1.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbObject1.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Get root Objects
	resultset, err = d.GetRootObjects(pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows < (originalTotalRows + 1) {
		t.Error("expected an increase in objects at root")
	}

	// Delete the object
	err = d.DeleteObject(dbObject1, true)
	if err != nil {
		t.Error(err)
	}

	// Get root Objects
	resultset, err = d.GetRootObjects(pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows {
		t.Error("expected same number of objects as before the test")
	}
}

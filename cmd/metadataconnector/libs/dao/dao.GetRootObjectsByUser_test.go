package dao_test

import (
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetRootObjectsByUser(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	user1 := models.ODUser{DistinguishedName: usernames[1]} // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	user2 := models.ODUser{DistinguishedName: usernames[2]} // "CN=test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	pagingRequest := protocol.PagingRequest{PageNumber: 1, PageSize: 1}
	// Get root Objects
	resultset, err := d.GetRootObjectsByUser(user1, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	// capture how many objects are rooted before changes
	originalTotalRows1 := resultset.TotalRows
	// The same for user2
	resultset, err = d.GetRootObjectsByUser(user2, pagingRequest)
	if err != nil {
		t.Failed()
	}
	originalTotalRows2 := resultset.TotalRows

	// Create an object with no parent under user1
	var object1 models.ODObject
	object1.Name = "Test GetRootObjectsByUser for user1"
	object1.CreatedBy = user1.DistinguishedName
	object1.TypeName.String = "Test Type"
	object1.TypeName.Valid = true
	object1.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions1 := make([]models.ODObjectPermission, 1)
	permissions1[0].CreatedBy = user1.DistinguishedName
	permissions1[0].Grantee = user1.DistinguishedName
	permissions1[0].AllowCreate = true
	permissions1[0].AllowRead = true
	permissions1[0].AllowUpdate = true
	permissions1[0].AllowDelete = true
	object1.Permissions = permissions1
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

	// Create an object with no parent under user2
	var object2 models.ODObject
	object2.Name = "Test GetRootObjectsByUser for user2"
	object2.CreatedBy = user2.DistinguishedName
	object2.TypeName.String = "Test Type"
	object2.TypeName.Valid = true
	object2.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions2 := make([]models.ODObjectPermission, 1)
	permissions2[0].CreatedBy = user2.DistinguishedName
	permissions2[0].Grantee = user2.DistinguishedName
	permissions2[0].AllowCreate = true
	permissions2[0].AllowRead = true
	permissions2[0].AllowUpdate = true
	permissions2[0].AllowDelete = true
	object2.Permissions = permissions2
	dbObject2, err := d.CreateObject(&object2)
	if err != nil {
		t.Error(err)
	}
	if dbObject2.ID == nil {
		t.Error("expected ID to be set")
	}
	if dbObject2.ModifiedBy != object2.CreatedBy {
		t.Error("expected ModifiedBy to match CreatedBy")
	}
	if dbObject2.TypeID == nil {
		t.Error("expected TypeID to be set")
	}

	// Get root Objects again
	resultset, err = d.GetRootObjectsByUser(user1, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != (originalTotalRows1 + 1) {
		t.Error("expected an increase in objects at root")
	}
	resultset, err = d.GetRootObjectsByUser(user2, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != (originalTotalRows2 + 1) {
		t.Error("expected an increase in objects at root")
	}

	// Delete the objects
	err = d.DeleteObject(dbObject1, true)
	if err != nil {
		t.Error(err)
	}
	err = d.DeleteObject(dbObject2, true)
	if err != nil {
		t.Error(err)
	}

	// Get root Objects again
	resultset, err = d.GetRootObjectsByUser(user1, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows1 {
		t.Error("expected same number of objects as before the test")
	}
	resultset, err = d.GetRootObjectsByUser(user2, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows != originalTotalRows2 {
		t.Error("expected same number of objects as before the test")
	}
}

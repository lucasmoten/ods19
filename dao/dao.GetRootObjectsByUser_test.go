package dao_test

import (
	"fmt"
	"testing"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetRootObjectsByUser(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	user1 := models.ODUser{DistinguishedName: usernames[1]}
	user2 := models.ODUser{DistinguishedName: usernames[2]}
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1}
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
	permissions1[0].Grantee = models.AACFlatten(user1.DistinguishedName)
	permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions1[0].CreatedBy)
	permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
	permissions1[0].AcmGrantee.UserDistinguishedName.String = permissions1[0].CreatedBy
	permissions1[0].AcmGrantee.UserDistinguishedName.Valid = true
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
	permissions2[0].Grantee = models.AACFlatten(user2.DistinguishedName)
	permissions2[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions2[0].CreatedBy)
	permissions2[0].AcmGrantee.Grantee = permissions2[0].Grantee
	permissions2[0].AcmGrantee.UserDistinguishedName.String = permissions2[0].CreatedBy
	permissions2[0].AcmGrantee.UserDistinguishedName.Valid = true
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
	if resultset.TotalRows <= originalTotalRows1 {
		t.Error("expected an increase in objects at root")
	}
	resultset, err = d.GetRootObjectsByUser(user2, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows <= originalTotalRows2 {
		t.Error("expected an increase in objects at root")
	}
}

func TestDAOGetRootObjectsForBobbyTables(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	bobbyTablesDN := "cn=bobby 'tables,o=theorg,ou=organizational unit,ou=people,c=us"
	user1 := models.ODUser{DistinguishedName: bobbyTablesDN}
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1}
	// Get root Objects
	resultset, err := d.GetRootObjectsByUser(user1, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	// capture how many objects are rooted before changes
	originalTotalRows1 := resultset.TotalRows

	// Create an object with no parent for bobby 'tables
	var object1 models.ODObject
	object1.Name = "Test GetRootObjectsByUser for bobby 'tables"
	object1.CreatedBy = user1.DistinguishedName
	object1.TypeName.String = "Test Type"
	object1.TypeName.Valid = true
	object1.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions1 := make([]models.ODObjectPermission, 1)
	permissions1[0].CreatedBy = user1.DistinguishedName
	permissions1[0].Grantee = models.AACFlatten(user1.DistinguishedName)
	permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions1[0].CreatedBy)
	permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
	permissions1[0].AcmGrantee.UserDistinguishedName.String = permissions1[0].CreatedBy
	permissions1[0].AcmGrantee.UserDistinguishedName.Valid = true
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

	// Get root Objects again
	resultset, err = d.GetRootObjectsByUser(user1, pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows <= originalTotalRows1 {
		t.Error("expected an increase in objects at root")
	}

}

package dao_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/server"
)

func TestDAOGetRootObjectsByUser(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1}
	// Get root Objects
	resultset, err := d.GetRootObjectsByUser(users[1], pagingRequest)
	if err != nil {
		t.Error(err)
	}
	// capture how many objects are rooted before changes
	originalTotalRows1 := resultset.TotalRows
	// The same for user2
	resultset, err = d.GetRootObjectsByUser(users[2], pagingRequest)
	if err != nil {
		t.Failed()
	}
	originalTotalRows2 := resultset.TotalRows

	// Create an object with no parent under user1
	var object1 models.ODObject
	object1.Name = "Test GetRootObjectsByUser for user1"
	object1.CreatedBy = users[1].DistinguishedName
	object1.TypeName = models.ToNullString("Test Type")
	acmUforTP1 := server.ValidACMUnclassified
	acmUforTP1 = strings.Replace(acmUforTP1, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s"]`, models.AACFlatten(usernames[1])), -1)
	object1.RawAcm = models.ToNullString(acmUforTP1)
	permissions1 := make([]models.ODObjectPermission, 1)
	permissions1[0].CreatedBy = users[1].DistinguishedName
	permissions1[0].Grantee = models.AACFlatten(users[1].DistinguishedName)
	permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, users[1].DistinguishedName)
	permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
	permissions1[0].AcmGrantee.ResourceString = models.ToNullString("user/" + users[1].DistinguishedName)
	permissions1[0].AcmGrantee.UserDistinguishedName = models.ToNullString(users[1].DistinguishedName)
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
	object2.CreatedBy = users[2].DistinguishedName
	object2.TypeName = models.ToNullString("Test Type")
	acmUforTP2 := server.ValidACMUnclassified
	acmUforTP2 = strings.Replace(acmUforTP2, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s"]`, models.AACFlatten(usernames[2])), -1)
	object2.RawAcm = models.ToNullString(acmUforTP2)
	permissions2 := make([]models.ODObjectPermission, 1)
	permissions2[0].CreatedBy = users[2].DistinguishedName
	permissions2[0].Grantee = models.AACFlatten(users[2].DistinguishedName)
	permissions2[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, users[2].DistinguishedName)
	permissions2[0].AcmGrantee.Grantee = permissions2[0].Grantee
	permissions2[0].AcmGrantee.ResourceString = models.ToNullString("user/" + users[2].DistinguishedName)
	permissions2[0].AcmGrantee.UserDistinguishedName = models.ToNullString(users[2].DistinguishedName)
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
	resultset, err = d.GetRootObjectsByUser(users[1], pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows <= originalTotalRows1 {
		t.Error("expected an increase in objects at root for user1")
	}
	resultset, err = d.GetRootObjectsByUser(users[2], pagingRequest)
	if err != nil {
		t.Error(err)
	}
	if resultset.TotalRows <= originalTotalRows2 {
		t.Error("expected an increase in objects at root for user2")
	}
}

func TestDAOGetRootObjectsForBobbyTables(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	// Bobby Tables refers to usage of characters in strings that can break SQL calls if the SQL is not properly
	// escaping values when being built up dynamically.  The users[11] contains an apostrophe and will be used
	// in this and similar tests going forward
	user1 := users[11]
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 1}
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
	object1.TypeName = models.ToNullString("Test Type")
	acmUforBobbyTables := server.ValidACMUnclassified
	acmUforBobbyTables = strings.Replace(acmUforBobbyTables, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s"]`, models.AACFlatten(users[11].DistinguishedName)), -1)
	object1.RawAcm = models.ToNullString(acmUforBobbyTables)
	permissions1 := make([]models.ODObjectPermission, 1)
	permissions1[0].CreatedBy = user1.DistinguishedName
	permissions1[0].Grantee = models.AACFlatten(user1.DistinguishedName)
	permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions1[0].CreatedBy)
	permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
	permissions1[0].AcmGrantee.ResourceString = models.ToNullString("user/" + user1.DistinguishedName)
	permissions1[0].AcmGrantee.UserDistinguishedName = models.ToNullString(user1.DistinguishedName)
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

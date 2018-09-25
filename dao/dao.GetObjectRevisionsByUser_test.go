package dao_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
)

func TestDAOGetObjectRevisionsByUser(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// Make an object
	var object models.ODObject
	object.CreatedBy = users[1].DistinguishedName
	object.Name = "Test Object Revision"
	object.TypeName = models.ToNullString("Test Object")
	acmUforTP1TP2 := ValidACMUnclassified
	acmUforTP1TP2 = strings.Replace(acmUforTP1TP2, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s","%s"]`, models.AACFlatten(usernames[1]), models.AACFlatten(usernames[2])), -1)
	object.RawAcm = models.ToNullString(acmUforTP1TP2)
	permissions := make([]models.ODObjectPermission, 2)
	permissions[0].CreatedBy = object.CreatedBy
	permissions[0].Grantee = models.AACFlatten(object.CreatedBy)
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, object.CreatedBy)
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.ResourceString = models.ToNullString("user/" + object.CreatedBy)
	permissions[0].AcmGrantee.UserDistinguishedName = models.ToNullString(object.CreatedBy)
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	permissions[0].AllowShare = true
	permissions[1].CreatedBy = object.CreatedBy
	permissions[1].Grantee = models.AACFlatten(users[2].DistinguishedName)
	permissions[1].AcmShare = fmt.Sprintf(`{"users":[%s]}`, users[2].DistinguishedName)
	permissions[1].AcmGrantee.Grantee = permissions[1].Grantee
	permissions[1].AcmGrantee.ResourceString = models.ToNullString("user/" + users[2].DistinguishedName)
	permissions[1].AcmGrantee.UserDistinguishedName = models.ToNullString(users[2].DistinguishedName)
	permissions[1].AllowCreate = true
	permissions[1].AllowRead = true
	permissions[1].AllowUpdate = true
	object.Permissions = permissions
	objectType, err := d.GetObjectTypeByName(object.TypeName.String, true, object.CreatedBy)
	if err != nil {
		t.Error(err)
	} else {
		object.TypeID = objectType.ID
	}
	dbObject, err := d.CreateObject(&object)
	if err != nil {
		t.Error("Failed to create object")
	}
	if dbObject.ID == nil {
		t.Error("Expected ID to be set")
	}
	object = dbObject
	t.Logf("object type id: %s", hex.EncodeToString(object.TypeID))
	ct1 := object.ChangeToken

	// Change it once
	object.Name = "Renamed by user 2"
	object.ModifiedBy = users[2].DistinguishedName
	err = d.UpdateObject(&object)
	if err != nil {
		t.Error("Failed to update object")
	}
	if object.ChangeCount != 1 {
		t.Error("expected change count to be 1")
	}
	ct2 := object.ChangeToken
	if strings.Compare(ct1, ct2) == 0 {
		t.Error("Change token did not change on first update")
	}

	// Change it twice
	object.Name = "Renamed again by user 1"
	object.ModifiedBy = users[1].DistinguishedName
	err = d.UpdateObject(&object)
	if err != nil {
		t.Error("Failed to update object")
	}
	if object.ChangeCount != 2 {
		t.Error("expected change count to be 2")
	}
	ct3 := object.ChangeToken
	if strings.Compare(ct2, ct3) == 0 {
		t.Error("Change token did not change on second update")
	}

	// Get list of revisions
	user := users[1]
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: dao.MaxPageSize, SortSettings: []dao.SortSetting{dao.SortSetting{SortField: "changecount", SortAscending: false}}}
	resultset, err := d.GetObjectRevisionsByUser(user, pagingRequest, object, false)
	if err != nil {
		t.Error("Error getting revisions for object")
	}
	if resultset.TotalRows != 3 {
		t.Errorf("Expected 3 revisions, got %d", resultset.TotalRows)
		t.Logf("objects in resultset: %d", len(resultset.Objects))
	}
	t.Logf("Object ID: %s", hex.EncodeToString(object.ID))
	for _, obj := range resultset.Objects {
		t.Logf("Object CC: %d, ModifiedBy: %s, Name: %s", obj.ChangeCount, obj.ModifiedBy, obj.Name)
	}
	if !t.Failed() && resultset.Objects[1].ModifiedBy != users[2].DistinguishedName {
		t.Errorf("Expected revision to be modified by %s, but got %s", users[2].DistinguishedName, resultset.Objects[1].ModifiedBy)
	}
	if !t.Failed() && resultset.Objects[0].Name != "Renamed again by user 1" {
		t.Errorf("Expected revision to be named %s, but got %s", "Renamed again by user 1", resultset.Objects[0].Name)
	}
	if !t.Failed() && resultset.Objects[2].Name != "Test Object Revision" {
		t.Errorf("Expected revision to be named %s, but got %s", "Test Object Revision", resultset.Objects[2].Name)
	}

	// Cleanup
	err = d.DeleteObject(user, object, true)
	if err != nil {
		t.Error(err)
	}

}

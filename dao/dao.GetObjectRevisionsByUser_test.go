package dao_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetObjectRevisionsByUser(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// Make an object
	var object models.ODObject
	object.CreatedBy = usernames[1]
	object.Name = "Test Object Revision"
	object.TypeName.String = "Test Object"
	object.TypeName.Valid = true
	object.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions := make([]models.ODObjectPermission, 2)
	permissions[0].CreatedBy = object.CreatedBy
	permissions[0].Grantee = models.AACFlatten(usernames[1])
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[1])
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.String = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.Valid = true
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	permissions[0].AllowShare = true
	permissions[1].CreatedBy = object.CreatedBy
	permissions[1].Grantee = models.AACFlatten(usernames[2])
	permissions[1].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[2])
	permissions[1].AcmGrantee.Grantee = permissions[1].Grantee
	permissions[1].AcmGrantee.UserDistinguishedName.String = permissions[1].Grantee
	permissions[1].AcmGrantee.UserDistinguishedName.Valid = true
	permissions[1].AllowCreate = true
	permissions[1].AllowRead = true
	permissions[1].AllowUpdate = true
	object.Permissions = permissions
	dbObject, err := d.CreateObject(&object)
	if err != nil {
		t.Error("Failed to create object")
	}
	if dbObject.ID == nil {
		t.Error("Expected ID to be set")
	}
	object = dbObject
	ct1 := object.ChangeToken

	// Change it once
	object.Name = "Renamed by user 2"
	object.ModifiedBy = usernames[2]
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
	object.ModifiedBy = usernames[1]
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
	user := models.ODUser{DistinguishedName: usernames[1]}
	pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: dao.MaxPageSize, SortSettings: []dao.SortSetting{dao.SortSetting{SortField: "changecount", SortAscending: false}}}
	resultset, err := d.GetObjectRevisionsByUser(user, pagingRequest, object, false)
	if err != nil {
		t.Error("Error getting revisions for object")
	}
	if resultset.TotalRows != 3 {
		t.Error(fmt.Errorf("Expected 3 revisions, got %d", resultset.TotalRows))
	}
	t.Logf("Object ID: %s", hex.EncodeToString(object.ID))
	for _, obj := range resultset.Objects {
		t.Logf("Object CC: %d, ModifiedBy: %s, Name: %s", obj.ChangeCount, obj.ModifiedBy, obj.Name)
	}
	if resultset.Objects[1].ModifiedBy != usernames[2] {
		t.Error(fmt.Errorf("Expected revision to be modified by %s, but got %s", usernames[2], resultset.Objects[1].ModifiedBy))
	}
	if resultset.Objects[0].Name != "Renamed again by user 1" {
		t.Error(fmt.Errorf("Expected revision to be named %s, but got %s", "Renamed again by user 1", resultset.Objects[0].Name))
	}
	if resultset.Objects[2].Name != "Test Object Revision" {
		t.Error(fmt.Errorf("Expected revision to be named %s, but got %s", "Test Object Revision", resultset.Objects[2].Name))
	}

	// Cleanup
	err = d.DeleteObject(user, object, true)
	if err != nil {
		t.Error(err)
	}

}

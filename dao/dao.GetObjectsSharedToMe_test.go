package dao_test

import (
	"bytes"
	"fmt"
	"testing"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

func TestDAOGetObjectsSharedToMe(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Logf("* Generating search prefix")
	searchPrefix, err := util.NewGUID()
	if err != nil {
		t.Logf("FAIL: Could not generate search prefix: %s", err.Error())
		t.Fail()
	}
	t.Logf("  prefix = %s", searchPrefix)

	t.Logf("* Make an object1 as user1, shared to user2")
	var object1 models.ODObject
	object1.CreatedBy = usernames[1]
	object1.Name = searchPrefix + " object1 (shared to user1)"
	object1.TypeName.String = "Test Object"
	object1.TypeName.Valid = true
	object1.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions1 := make([]models.ODObjectPermission, 2)
	permissions1[0].CreatedBy = object1.CreatedBy
	permissions1[0].Grantee = models.AACFlatten(usernames[1])
	permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[1])
	permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
	permissions1[0].AcmGrantee.UserDistinguishedName = models.ToNullString(usernames[1])
	permissions1[0].AllowCreate = true
	permissions1[0].AllowRead = true
	permissions1[0].AllowUpdate = true
	permissions1[0].AllowDelete = true
	permissions1[0].AllowShare = true
	permissions1[1].CreatedBy = object1.CreatedBy
	permissions1[1].Grantee = models.AACFlatten(usernames[2])
	permissions1[1].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[2])
	permissions1[1].AcmGrantee.Grantee = permissions1[1].Grantee
	permissions1[1].AcmGrantee.UserDistinguishedName = models.ToNullString(usernames[2])
	permissions1[1].AllowRead = true
	object1.Permissions = permissions1
	createdObject1, err := d.CreateObject(&object1)
	if err != nil {
		t.Error("Failed to create object")
	}
	if createdObject1.ID == nil {
		t.Error("Expected ID to be set")
	}
	object1 = createdObject1

	t.Logf("* Make an object2 as user1, shared to everyone")
	var object2 models.ODObject
	object2.CreatedBy = usernames[1]
	object2.Name = searchPrefix + " object2 (shared to everyone)"
	object2.TypeName.String = "Test Object"
	object2.TypeName.Valid = true
	object2.RawAcm.String = testhelpers.ValidACMUnclassified
	permissions2 := make([]models.ODObjectPermission, 2)
	permissions2[0].CreatedBy = object1.CreatedBy
	permissions2[0].Grantee = models.AACFlatten(usernames[1])
	permissions2[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[1])
	permissions2[0].AcmGrantee.Grantee = permissions2[0].Grantee
	permissions2[0].AcmGrantee.UserDistinguishedName = models.ToNullString(usernames[1])
	permissions2[0].AllowCreate = true
	permissions2[0].AllowRead = true
	permissions2[0].AllowUpdate = true
	permissions2[0].AllowDelete = true
	permissions2[0].AllowShare = true
	permissions2[1].CreatedBy = object1.CreatedBy
	permissions2[1].Grantee = models.AACFlatten(models.EveryoneGroup)
	permissions2[1].AcmShare = fmt.Sprintf(`{"projects":{"%s":{"disp_nm":"%s","groups":["%s"]}}}`, "", "", models.EveryoneGroup)
	permissions2[1].AcmGrantee.Grantee = permissions2[1].Grantee
	permissions2[1].AcmGrantee.GroupName = models.ToNullString(models.EveryoneGroup)
	permissions2[1].AllowRead = true
	object2.Permissions = permissions2
	createdObject2, err := d.CreateObject(&object2)
	if err != nil {
		t.Error("Failed to create object")
	}
	if createdObject2.ID == nil {
		t.Error("Expected ID to be set")
	}
	object2 = createdObject2

	paging := protocol.PagingRequest{}
	paging.PageNumber = 1
	paging.PageSize = 1000
	filter := []protocol.FilterSetting{}
	filter1 := protocol.FilterSetting{}
	filter1.FilterField = "name"
	filter1.Condition = "contains"
	filter1.Expression = searchPrefix
	filter = append(filter, filter1)
	paging.FilterSettings = filter

	t.Logf("* Get objects shared to me as user1 containing %s", searchPrefix)
	user1 := models.ODUser{}
	user1.DistinguishedName = usernames[1]
	user1Snippets, err := acm.NewODriveRawSnippetFieldsFromSnippetResponse(SnippetDAOTP01)
	if err != nil {
		t.Logf("FAIL: Error converting snippets for user1 %s", err.Error())
		t.Fail()
	}
	user1.Snippets = &user1Snippets
	user1object1Found := false
	user1object2Found := false
	user1SharedToMe, err := d.GetObjectsSharedToMe(user1, paging)
	t.Logf("  total rows = %d", user1SharedToMe.TotalRows)
	for _, user1object := range user1SharedToMe.Objects {
		t.Logf("  %s is shared to user1", user1object.Name)
		if bytes.Equal(user1object.ID, object1.ID) {
			user1object1Found = true
		}
		if bytes.Equal(user1object.ID, object2.ID) {
			user1object2Found = true
		}
	}
	if user1object1Found {
		t.Logf("FAIL: object1 created by user1 showed up in user1 sharedtome list")
		t.Fail()
	}
	if user1object2Found {
		t.Logf("FAIL: object2 created by user1 showed up in user1 sharedtome list")
		t.Fail()
	}

	t.Logf("* Get objects shared to me as user2 containing %s", searchPrefix)
	user2 := models.ODUser{}
	user2.DistinguishedName = usernames[2]
	user2Snippets, err := acm.NewODriveRawSnippetFieldsFromSnippetResponse(SnippetDAOTP02)
	if err != nil {
		t.Logf("FAIL: Error converting snippets for user2 %s", err.Error())
		t.Fail()
	}
	user2.Snippets = &user2Snippets
	user2object1Found := false
	user2object2Found := false
	user2SharedToMe, err := d.GetObjectsSharedToMe(user2, paging)
	t.Logf("  total rows = %d", user2SharedToMe.TotalRows)
	for _, user2object := range user2SharedToMe.Objects {
		t.Logf("  %s is shared to user2", user2object.Name)
		if bytes.Equal(user2object.ID, object1.ID) {
			user2object1Found = true
		}
		if bytes.Equal(user2object.ID, object2.ID) {
			user2object2Found = true
		}
	}
	if !user2object1Found {
		t.Logf("FAIL: object1 created by user1 did not show up in user2 sharedtome list")
		t.Fail()
	}
	if user2object2Found {
		t.Logf("FAIL: object2 created by user1 showed up in user2 sharedtome list")
		t.Fail()
	}

}

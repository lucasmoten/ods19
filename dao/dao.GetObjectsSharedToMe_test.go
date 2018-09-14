package dao_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/server"
	"bitbucket.di2e.net/dime/object-drive-server/util"
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
	object1.CreatedBy = users[1].DistinguishedName
	object1.Name = searchPrefix + " object1 (shared to user1)"
	object1.TypeName = models.ToNullString("Test Object")
	acmUforTP1TP2 := server.ValidACMUnclassified
	acmUforTP1TP2 = strings.Replace(acmUforTP1TP2, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s","%s"]`, models.AACFlatten(usernames[1]), models.AACFlatten(usernames[2])), -1)
	object1.RawAcm = models.ToNullString(acmUforTP1TP2)
	permissions1 := make([]models.ODObjectPermission, 2)
	permissions1[0].CreatedBy = object1.CreatedBy
	permissions1[0].Grantee = models.AACFlatten(object1.CreatedBy)
	permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, object1.CreatedBy)
	permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
	permissions1[0].AcmGrantee.ResourceString = models.ToNullString("user/" + object1.CreatedBy)
	permissions1[0].AcmGrantee.UserDistinguishedName = models.ToNullString(object1.CreatedBy)
	permissions1[0].AllowCreate = true
	permissions1[0].AllowRead = true
	permissions1[0].AllowUpdate = true
	permissions1[0].AllowDelete = true
	permissions1[0].AllowShare = true
	permissions1[1].CreatedBy = object1.CreatedBy
	permissions1[1].Grantee = models.AACFlatten(users[2].DistinguishedName)
	permissions1[1].AcmShare = fmt.Sprintf(`{"users":[%s]}`, users[2].DistinguishedName)
	permissions1[1].AcmGrantee.Grantee = permissions1[1].Grantee
	permissions1[1].AcmGrantee.ResourceString = models.ToNullString("user/" + users[2].DistinguishedName)
	permissions1[1].AcmGrantee.UserDistinguishedName = models.ToNullString(users[2].DistinguishedName)
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
	object2.CreatedBy = users[1].DistinguishedName
	object2.Name = searchPrefix + " object2 (shared to everyone)"
	object2.TypeName = models.ToNullString("Test Object")
	object2.RawAcm.String = server.ValidACMUnclassified
	permissions2 := make([]models.ODObjectPermission, 2)
	permissions2[0].CreatedBy = object2.CreatedBy
	permissions2[0].Grantee = models.AACFlatten(object2.CreatedBy)
	permissions2[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, object2.CreatedBy)
	permissions2[0].AcmGrantee.Grantee = permissions2[0].Grantee
	permissions2[0].AcmGrantee.ResourceString = models.ToNullString("user/" + object2.CreatedBy)
	permissions2[0].AcmGrantee.UserDistinguishedName = models.ToNullString(object2.CreatedBy)
	permissions2[0].AllowCreate = true
	permissions2[0].AllowRead = false
	permissions2[0].AllowUpdate = true
	permissions2[0].AllowDelete = true
	permissions2[0].AllowShare = true
	permissions2[1].CreatedBy = object2.CreatedBy
	permissions2[1].Grantee = models.AACFlatten(models.EveryoneGroup)
	permissions2[1].AcmShare = fmt.Sprintf(`{"projects":{"%s":{"disp_nm":"%s","groups":["%s"]}}}`, "", "", models.EveryoneGroup)
	permissions2[1].AcmGrantee.Grantee = permissions2[1].Grantee
	permissions2[1].AcmGrantee.ResourceString = models.ToNullString("group/" + models.EveryoneGroup)
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

	paging := dao.PagingRequest{}
	paging.PageNumber = 1
	paging.PageSize = 1000
	filter := []dao.FilterSetting{}
	filter1 := dao.FilterSetting{}
	filter1.FilterField = "name"
	filter1.Condition = "contains"
	filter1.Expression = searchPrefix
	filter = append(filter, filter1)
	paging.FilterSettings = filter

	t.Logf("* Get objects shared to me as user1 containing %s", searchPrefix)
	user1object1Found := false
	user1object2Found := false
	user1SharedToMe, err := d.GetObjectsSharedToMe(users[1], paging)
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
	user2object1Found := false
	user2object2Found := false
	user2SharedToMe, err := d.GetObjectsSharedToMe(users[2], paging)
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

func TestDAOGetObjectsSharedToMeWithApostropheInDN595(t *testing.T) {
	//t.SkipNow()

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

	t.Logf("* Make an object1 as user1, shared to user 11 with apostrophe in DN")
	var object1 models.ODObject
	object1.CreatedBy = users[1].DistinguishedName
	object1.Name = searchPrefix + " object1 (shared to user11)"
	object1.TypeName = models.ToNullString("Test Object")
	acmUforTP1TP11 := server.ValidACMUnclassified
	acmUforTP1TP11 = strings.Replace(acmUforTP1TP11, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s","%s"]`, models.AACFlatten(object1.CreatedBy), models.AACFlatten(users[11].DistinguishedName)), -1)
	object1.RawAcm = models.ToNullString(acmUforTP1TP11)
	permissions1 := make([]models.ODObjectPermission, 2)
	permissions1[0].CreatedBy = object1.CreatedBy
	permissions1[0].Grantee = models.AACFlatten(object1.CreatedBy)
	permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, object1.CreatedBy)
	permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
	permissions1[0].AcmGrantee.ResourceString = models.ToNullString("user/" + object1.CreatedBy)
	permissions1[0].AcmGrantee.UserDistinguishedName = models.ToNullString(object1.CreatedBy)
	permissions1[0].AllowCreate = true
	permissions1[0].AllowRead = true
	permissions1[0].AllowUpdate = true
	permissions1[0].AllowDelete = true
	permissions1[0].AllowShare = true
	permissions1[1].CreatedBy = object1.CreatedBy
	permissions1[1].Grantee = models.AACFlatten(users[11].DistinguishedName)
	permissions1[1].AcmShare = fmt.Sprintf(`{"users":[%s]}`, users[11].DistinguishedName)
	permissions1[1].AcmGrantee.Grantee = permissions1[1].Grantee
	permissions1[1].AcmGrantee.ResourceString = models.ToNullString("user/" + users[11].DistinguishedName)
	permissions1[1].AcmGrantee.UserDistinguishedName = models.ToNullString(users[11].DistinguishedName)
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

	t.Logf("* Setting up search filter")
	paging := dao.PagingRequest{}
	paging.PageNumber = 1
	paging.PageSize = 1000
	filter := []dao.FilterSetting{}
	filter1 := dao.FilterSetting{}
	filter1.FilterField = "name"
	filter1.Condition = "contains"
	filter1.Expression = searchPrefix
	filter = append(filter, filter1)
	paging.FilterSettings = filter

	t.Logf("* Get objects shared to me as user11 containing %s", searchPrefix)
	user11object1Found := false
	// Short delay to accomodate the async call for useracm association
	time.Sleep(time.Millisecond * 250)
	user11SharedToMe, err := d.GetObjectsSharedToMe(users[11], paging)
	t.Logf("  total rows = %d", user11SharedToMe.TotalRows)
	for _, user11object := range user11SharedToMe.Objects {
		t.Logf("  %s is shared to user11", user11object.Name)
		if bytes.Equal(user11object.ID, object1.ID) {
			user11object1Found = true
		}
	}
	if !user11object1Found {
		t.Logf("FAIL: object1 created by user1 did not show up in user11 sharedtome list")
		t.Fail()
	}
}

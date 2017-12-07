package dao_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/server"
)

func TestDAOGetChildObjectsByUser(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var parent models.ODObject
	var child1 models.ODObject
	var child2 models.ODObject

	// Create our parent object
	parent.Name = "Test GetChildObjectsByUser Parent"
	parent.CreatedBy = usernames[1]
	parent.TypeName.String = "Test Type"
	parent.TypeName.Valid = true
	parent.RawAcm.String = server.ValidACMUnclassified
	// NEW! Add permissions...
	permissions := make([]models.ODObjectPermission, 2)
	permissions[0].CreatedBy = parent.CreatedBy
	permissions[0].Grantee = models.AACFlatten(usernames[1])
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[1])
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.ResourceString = models.ToNullString("user/" + usernames[1])
	permissions[0].AcmGrantee.UserDistinguishedName = models.ToNullString(usernames[1])
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	permissions[0].AllowShare = true
	permissions[1].CreatedBy = parent.CreatedBy
	permissions[1].Grantee = models.AACFlatten(usernames[2])
	permissions[1].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[2])
	permissions[1].AcmGrantee.Grantee = permissions[1].Grantee
	permissions[1].AcmGrantee.ResourceString = models.ToNullString("user/" + usernames[2])
	permissions[1].AcmGrantee.UserDistinguishedName = models.ToNullString(usernames[2])
	permissions[1].AllowCreate = true
	permissions[1].AllowRead = true
	parent.Permissions = permissions
	dbParent, err := d.CreateObject(&parent)
	if err != nil {
		t.Error(err)
	} else {
		if dbParent.ID == nil {
			t.Error("expected ID to be set")
		}
		if dbParent.ModifiedBy != parent.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if dbParent.TypeID == nil {
			t.Error("expected TypeID to be set")
		}

		// Create our child object from TP1
		child1.Name = "Test GetChildObjectsByUser Child by TP1"
		child1.CreatedBy = usernames[1]
		child1.ParentID = dbParent.ID
		child1.TypeName.String = "Test Type"
		child1.TypeName.Valid = true
		acmUforTP1 := server.ValidACMUnclassified
		acmUforTP1 = strings.Replace(acmUforTP1, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s"]`, models.AACFlatten(usernames[1])), -1)
		child1.RawAcm = models.ToNullString(acmUforTP1)
		// NEW! Add permissions...
		permissions1 := make([]models.ODObjectPermission, 1)
		permissions1[0].CreatedBy = child1.CreatedBy
		permissions1[0].Grantee = models.AACFlatten(child1.CreatedBy)
		permissions1[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, child1.CreatedBy)
		permissions1[0].AcmGrantee.Grantee = permissions1[0].Grantee
		permissions1[0].AcmGrantee.ResourceString = models.ToNullString("user/" + child1.CreatedBy)
		permissions1[0].AcmGrantee.UserDistinguishedName = models.ToNullString(child1.CreatedBy)
		permissions1[0].AllowCreate = true
		permissions1[0].AllowRead = true
		permissions1[0].AllowUpdate = true
		permissions1[0].AllowDelete = true
		permissions1[0].AllowShare = true
		child1.Permissions = permissions1
		dbChild1, err := d.CreateObject(&child1)
		if err != nil {
			t.Error(err)
		}
		if dbChild1.ID == nil {
			t.Error("expected ID to be set")
		}
		if dbChild1.ModifiedBy != child1.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if dbChild1.TypeID == nil {
			t.Error("expected TypeID to be set")
		}
		if !bytes.Equal(dbChild1.ParentID, dbParent.ID) {
			t.Error("expected child parentID to match parent ID")
		}

		// Create our child object from TP2
		child2.Name = "Test GetChildObjectsByUser Child by TP2"
		child2.CreatedBy = usernames[2]
		child2.ParentID = dbParent.ID
		child2.TypeName = models.ToNullString("Test Type")
		acmUforTP1TP2 := server.ValidACMUnclassified
		acmUforTP1TP2 = strings.Replace(acmUforTP1TP2, `"f_share":[]`, fmt.Sprintf(`"f_share":["%s","%s"]`, models.AACFlatten(usernames[1]), models.AACFlatten(usernames[2])), -1)
		child2.RawAcm = models.ToNullString(acmUforTP1TP2)
		// NEW! Add permissions...
		permissions2 := make([]models.ODObjectPermission, 2)
		permissions2[0].CreatedBy = child2.CreatedBy
		permissions2[0].Grantee = models.AACFlatten(permissions2[0].CreatedBy)
		permissions2[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions2[0].CreatedBy)
		permissions2[0].AcmGrantee.Grantee = permissions2[0].Grantee
		permissions2[0].AcmGrantee.ResourceString = models.ToNullString("user/" + usernames[1])
		permissions2[0].AcmGrantee.UserDistinguishedName = models.ToNullString(permissions2[0].CreatedBy)
		permissions2[0].AllowCreate = true
		permissions2[0].AllowRead = true
		permissions2[0].AllowUpdate = true
		permissions2[0].AllowDelete = true
		permissions2[0].AllowShare = true
		permissions2[1].CreatedBy = child2.CreatedBy
		permissions2[1].Grantee = models.AACFlatten(usernames[2])
		permissions2[1].AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[2])
		permissions2[1].AcmGrantee.Grantee = permissions2[1].Grantee
		permissions2[1].AcmGrantee.ResourceString = models.ToNullString("user/" + usernames[2])
		permissions2[1].AcmGrantee.UserDistinguishedName = models.ToNullString(usernames[2])
		permissions2[1].AllowCreate = true
		permissions2[1].AllowRead = true
		child2.Permissions = permissions2
		dbChild2, err := d.CreateObject(&child2)
		if err != nil {
			t.Error(err)
		}
		if dbChild2.ID == nil {
			t.Error("expected ID to be set")
		}
		if dbChild2.ModifiedBy != child2.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if dbChild2.TypeID == nil {
			t.Error("expected TypeID to be set")
		}
		if !bytes.Equal(dbChild2.ParentID, dbParent.ID) {
			t.Error("expected child parentID to match parent ID")
		}
		user := users[2]
		pagingRequest := dao.PagingRequest{PageNumber: 1, PageSize: 10}
		resultset, err := d.GetChildObjectsByUser(user, pagingRequest, dbParent)
		if err != nil {
			t.Error(err)
		}
		if resultset.TotalRows != 1 {
			t.Error(fmt.Errorf("Resultset had %d totalrows", resultset.TotalRows))
			t.Error("expected 1 child")
			for i, o := range resultset.Objects {
				t.Errorf("%d = %s", i, o.Name)
			}
		} else {
			if resultset.Objects[0].ModifiedBy != child2.CreatedBy {
				t.Error("expected result modifiedBy to match child2 created by")
			}
			if !strings.Contains(resultset.Objects[0].ModifiedBy, "tester02") {
				t.Error("expected result ModifiedBy to be by tester02")
			}
		}
	}

	//cleanup
	user := models.ODUser{DistinguishedName: dbParent.CreatedBy}
	err = d.DeleteObject(user, dbParent, true)
	if err != nil {
		t.Error(err)
	}
}

package dao_test

import (
	"bytes"
	"strings"
	"testing"

	"decipher.com/oduploader/metadata/models"
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
	parent.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	parent.TypeName.String = "Test Type"
	parent.TypeName.Valid = true
	// NEW! Add permissions...
	permissions := make([]models.ODObjectPermission, 2)
	permissions[0].CreatedBy = parent.CreatedBy
	permissions[0].Grantee = usernames[1]
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	permissions[1].CreatedBy = parent.CreatedBy
	permissions[1].Grantee = usernames[2]
	permissions[1].AllowCreate = true
	permissions[1].AllowRead = true
	parent.Permissions = permissions
	err := d.CreateObject(&parent, nil)
	if err != nil {
		t.Error(err)
	} else {
		if parent.ID == nil {
			t.Error("expected ID to be set")
		}
		if parent.ModifiedBy != parent.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if parent.TypeID == nil {
			t.Error("expected TypeID to be set")
		}

		// Create our child object from TP1
		child1.Name = "Test GetChildObjectsByUser Child by TP1"
		child1.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		child1.ParentID = parent.ID
		child1.TypeName.String = "Test Type"
		child1.TypeName.Valid = true
		// NEW! Add permissions...
		permissions1 := make([]models.ODObjectPermission, 2)
		permissions1[0].CreatedBy = child1.CreatedBy
		permissions1[0].Grantee = usernames[1]
		permissions1[0].AllowCreate = true
		permissions1[0].AllowRead = true
		permissions1[0].AllowUpdate = true
		permissions1[0].AllowDelete = true
		permissions1[1].CreatedBy = child1.CreatedBy
		permissions1[1].Grantee = usernames[2]
		permissions1[1].AllowCreate = true
		permissions1[1].AllowRead = true
		child1.Permissions = permissions1
		err = d.CreateObject(&child1, nil)
		if err != nil {
			t.Error(err)
		}
		if child1.ID == nil {
			t.Error("expected ID to be set")
		}
		if child1.ModifiedBy != child1.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if child1.TypeID == nil {
			t.Error("expected TypeID to be set")
		}
		if !bytes.Equal(child1.ParentID, parent.ID) {
			t.Error("expected child parentID to match parent ID")
		}

		// Create our child object from TP2
		child2.Name = "Test GetChildObjectsByUser Child by TP2"
		child2.CreatedBy = usernames[2] // "CN=test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
		child2.ParentID = parent.ID
		child2.TypeName.String = "Test Type"
		child2.TypeName.Valid = true
		// NEW! Add permissions...
		permissions2 := make([]models.ODObjectPermission, 2)
		permissions2[0].CreatedBy = child2.CreatedBy
		permissions2[0].Grantee = usernames[1]
		permissions2[0].AllowCreate = true
		permissions2[0].AllowRead = true
		permissions2[0].AllowUpdate = true
		permissions2[0].AllowDelete = true
		permissions2[1].CreatedBy = child2.CreatedBy
		permissions2[1].Grantee = usernames[2]
		permissions2[1].AllowCreate = true
		permissions2[1].AllowRead = true
		child2.Permissions = permissions2
		err = d.CreateObject(&child2, nil)
		if err != nil {
			t.Error(err)
		}
		if child2.ID == nil {
			t.Error("expected ID to be set")
		}
		if child2.ModifiedBy != child2.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if child2.TypeID == nil {
			t.Error("expected TypeID to be set")
		}
		if !bytes.Equal(child2.ParentID, parent.ID) {
			t.Error("expected child parentID to match parent ID")
		}
		resultset, err := d.GetChildObjectsByUser("", 1, 10, &parent, child2.CreatedBy)
		if err != nil {
			t.Error(err)
		}
		if resultset.TotalRows != 1 {
			t.Error("expected 1 child")
		} else {
			if resultset.Objects[0].ModifiedBy != child2.CreatedBy {
				t.Error("expected result modifiedBy to match child2 created by")
			}
			if !strings.Contains(resultset.Objects[0].ModifiedBy, "tester02") {
				t.Error("expected result ModifiedBy to be by tester02")
			}
		}
	}

	// cleanup
	// err = d.DeleteObject(&parent, true)
	// if err != nil {
	// 	t.Error(err)
	// }
}

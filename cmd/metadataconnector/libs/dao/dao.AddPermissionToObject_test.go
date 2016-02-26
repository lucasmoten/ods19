package dao_test

import (
	"testing"

	"decipher.com/oduploader/metadata/models"
)

func TestDAOAddPermissionToObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var object models.ODObject

	// Create our parent object
	object.Name = "Test Object for Permissions"
	object.CreatedBy = usernames[1] // "CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US"
	object.TypeName.String = "Test Type"
	object.TypeName.Valid = true
	err := d.CreateObject(&object, nil)
	if err != nil {
		t.Error(err)
	} else {
		if object.ID == nil {
			t.Error("expected ID to be set")
		}
		if object.ModifiedBy != object.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if object.TypeID == nil {
			t.Error("expected TypeID to be set")
		}

		// Now add the permission
		permission := models.ODObjectPermission{}
		permission.CreatedBy = usernames[1]
		permission.Grantee = usernames[1]
		permission.AllowCreate = true
		permission.AllowRead = true
		permission.AllowUpdate = true
		permission.AllowDelete = true
		dbPermission, err := d.AddPermissionToObject(usernames[1], &object, &permission)
		if err != nil {
			t.Error(err)
		}
		if dbPermission.ID == nil {
			t.Error("expected ID to be set on permission")
		}
		if dbPermission.ModifiedBy != permission.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy for permission")
		}

	}
}

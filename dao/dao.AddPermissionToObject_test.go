package dao_test

import (
	"fmt"
	"os"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

func TestDAOAddPermissionToObject(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var object models.ODObject

	// Create our parent object
	object.Name = "Test Object for Permissions"
	object.CreatedBy = usernames[1]
	object.TypeName = models.ToNullString("Test Type")
	object.RawAcm = models.ToNullString(ValidACMUnclassified)
	objectType, err := d.GetObjectTypeByName(object.TypeName.String, true, object.CreatedBy)
	if err != nil {
		t.Error(err)
	} else {
		object.TypeID = objectType.ID
	}
	dbObject, err := d.CreateObject(&object)
	if err != nil {
		t.Error(err)
	} else {
		if dbObject.ID == nil {
			t.Error("expected ID to be set")
		}
		if dbObject.ModifiedBy != object.CreatedBy {
			t.Error("expected ModifiedBy to match CreatedBy")
		}
		if dbObject.TypeID == nil {
			t.Error("expected TypeID to be set")
		}

		masterkey, err := config.MaybeDecrypt(os.Getenv("OD_ENCRYPT_MASTERKEY"))
		if err != nil {
			t.Logf("unable to get encrypt key: %v", err)
			t.FailNow()
		}

		if len(masterkey) == 0 {
			// this is just a test. use something random.
			guid, _ := util.NewGUID()
			masterkey = guid
			// note that if you rely on these permissions later, it will do you no good.
		}

		// Now add the permission
		permission := models.ODObjectPermission{}
		permission.CreatedBy = usernames[1]
		permission.Grantee = models.AACFlatten(usernames[1])
		permission.AcmGrantee.ResourceString = models.ToNullString("user/" + usernames[1])
		permission.AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[1])
		permission.AcmGrantee.Grantee = permission.Grantee
		permission.AcmGrantee.ResourceString = models.ToNullString("user/" + usernames[1])
		permission.AcmGrantee.UserDistinguishedName = models.ToNullString(usernames[1])
		permission.AllowCreate = true
		permission.AllowRead = true
		permission.AllowUpdate = true
		permission.AllowDelete = true
		permission.AllowShare = true
		dbPermission, err := d.AddPermissionToObject(dbObject, &permission)
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

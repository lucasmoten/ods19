package dao_test

import (
	"fmt"
	"os"
	"testing"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
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
	object.RawAcm = models.ToNullString(testhelpers.ValidACMUnclassified)
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
		permission.Grantee = usernames[1]
		permission.AcmShare = fmt.Sprintf(`{"users":[%s]}`, usernames[1])
		permission.AcmGrantee.Grantee = permission.Grantee
		permission.AcmGrantee.UserDistinguishedName.String = permission.Grantee
		permission.AcmGrantee.UserDistinguishedName.Valid = true
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

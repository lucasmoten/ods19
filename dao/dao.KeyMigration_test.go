package dao_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/crypto"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
)

//The migration and key rotation functions rely on
//stored functions that perform the same crypto operations
//as in the Go code.  This is testing that they actually perform
//the same operation, to demonstrate that migration and rotation of keys
//will succeed.
func TestDAOKeyMigrateRotate(t *testing.T) {
	//The existing masterkey
	m := "otterpaws"
	t.Logf("masterKey: %s", m)

	//The permission
	p := models.ODObjectPermission{Grantee: "cn=testing"}
	p.AllowCreate = true
	p.AllowRead = true
	p.AllowUpdate = true
	p.AllowDelete = true
	p.AllowShare = true

	//This is the key that the file is encrypted under
	fileKey := crypto.CreateKey()
	t.Logf("fileKey: %s", hex.EncodeToString(fileKey))

	//The IV for a new style permission
	newPermissionIV := crypto.CreatePermissionIV()
	t.Logf("newPermissionIV: %s", hex.EncodeToString(newPermissionIV))

	//New style encrypted keys look like this
	newEncryptedKey := crypto.ApplyPassphrase(m, newPermissionIV, fileKey)
	t.Logf("newEncryptedKey: %s", hex.EncodeToString(newEncryptedKey))

	//The signature over an object_permission is like this:
	newPermissionMAC := crypto.DoMAC(
		m,
		newPermissionIV,
		p.Grantee,
		p.AllowCreate,
		p.AllowRead,
		p.AllowUpdate,
		p.AllowDelete,
		p.AllowShare,
		newEncryptedKey,
	)
	t.Logf("newPermissionMAC: %s", hex.EncodeToString(newPermissionMAC))

	//Call the stored functions to ensure that we are getting correct results
	var result string
	tx := db.MustBegin()

	//Generate the correct new (after migration) key
	err := tx.Get(&result, `
        select 
            lcase(bitwise256_xor(
                new_keydecrypt(?,?),
                ?
            )) encryptKey
    `,
		m,
		hex.EncodeToString(newPermissionIV),
		hex.EncodeToString(fileKey),
	)
	if err != nil {
		t.Errorf("unable to invoke stored function: %v", err)
	}
	t.Logf("migrated encryptedKey: %s", result)

	if strings.Compare(result, hex.EncodeToString(newEncryptedKey)) != 0 {
		t.Error("migrated to wrong key")
	}

	//Generate the correct (migrated) mac
	err = tx.Get(&result, `
        select lcase( 
            new_keymac(
                ?, 
                ?, 
                ?, 
                ?, 
                ?, 
                ?, 
                ?, 
                ?
            ) 
        ) mac   `,
		m,
		p.Grantee,
		p.AllowCreate,
		p.AllowRead,
		p.AllowUpdate,
		p.AllowDelete,
		p.AllowShare,
		hex.EncodeToString(newEncryptedKey),
	)
	if err != nil {
		t.Errorf("unable to invoke stored function: %v", err)
	}
	t.Logf("migrated permissionMAC: %s", result)

	if strings.Compare(result, hex.EncodeToString(newPermissionMAC)) != 0 {
		t.Error("migrated to wrong mac")
	}

	//Now, rotate these keys to a new value
	m2 := "asdfjklqwer"
	t.Logf("newMasterKey: %s", m2)

	//We are expecting the original key encrypted under m2
	//NOTE: we didn't change the IV, but we did change (m,IV),
	// so that is safe, particularly because fileKey never changes.
	rotatedEncryptedKey := crypto.ApplyPassphrase(m2, newPermissionIV, fileKey)
	t.Logf("rotatedEncryptedKey: %s", hex.EncodeToString(rotatedEncryptedKey))

	//The signature over an object_permission is like this:
	rotatedPermissionMAC := crypto.DoMAC(
		m2,
		newPermissionIV,
		p.Grantee,
		p.AllowCreate,
		p.AllowRead,
		p.AllowUpdate,
		p.AllowDelete,
		p.AllowShare,
		rotatedEncryptedKey,
	)
	t.Logf("rotatedPermissionMAC: %s", hex.EncodeToString(rotatedPermissionMAC))

	//Generate the correct rotated key
	err = tx.Get(&result, `
    select lcase(bitwise256_xor( 
        new_keydecrypt( ?, ?), 
        bitwise256_xor(
            new_keydecrypt(?, ?), 
            ?
        )
    )) rotatedEncryptedKey
    `,
		m2,
		hex.EncodeToString(newPermissionIV),
		m,
		hex.EncodeToString(newPermissionIV),
		hex.EncodeToString(newEncryptedKey),
	)
	if err != nil {
		t.Errorf("unable to invoke stored function: %v", err)
	}
	t.Logf("rotated encryptedKey: %s", result)

	if strings.Compare(result, hex.EncodeToString(rotatedEncryptedKey)) != 0 {
		t.Error("rotated to wrong key")
	}

	//Generate the rotated mac
	err = tx.Get(&result, `
        select lcase( 
            new_keymac(
                ?, 
                ?, 
                ?, 
                ?, 
                ?, 
                ?, 
                ?, 
                ?
            ) 
        ) mac   `,
		m2,
		p.Grantee,
		p.AllowCreate,
		p.AllowRead,
		p.AllowUpdate,
		p.AllowDelete,
		p.AllowShare,
		hex.EncodeToString(rotatedEncryptedKey),
	)
	if err != nil {
		t.Errorf("unable to invoke stored function: %v", err)
	}
	t.Logf("rotated permissionMAC: %s", result)

	if strings.Compare(result, hex.EncodeToString(rotatedPermissionMAC)) != 0 {
		t.Error("rotated to wrong mac")
	}

	tx.Commit()
}

package models

import "decipher.com/object-drive-server/cmd/odrive/libs/utils"

// ODObjectPermission is a nestable structure defining the attributes for
// permissions granted on an object for users who have access to the object
// in Object Drive
type ODObjectPermission struct {
	ODCommonMeta
	ODChangeTracking
	// ObjectID identifies the object for which this permission applies.
	ObjectID []byte `db:"objectId"`
	// Grantee indicates the user, identified by distinguishedName from the user
	// table for which this grant applies
	Grantee string `db:"grantee"`
	// AllowCreate indicates whether the grantee has permission to create child
	// objects beneath this object
	AllowCreate bool `db:"allowCreate"`
	// AllowRead indicates whether the grantee has permission to read this
	// object. This is the most fundamental permission granted, and should always
	// be true as only records need to exist where permissions are granted as
	// the system denies access by default. Read access to an object is necessary
	// to perform any other action on the object.
	AllowRead bool `db:"allowRead"`
	// AllowUpdate indicates whether the grantee has permission to update this
	// object
	AllowUpdate bool `db:"allowUpdate"`
	// AllowDelete indicates whether the grantee has permission to delete this
	// object
	AllowDelete bool `db:"allowDelete"`
	// AllowShare indicates whether the grantee has permission to view and
	// alter permissions on this object
	AllowShare bool `db:"allowShare"`
	// ExplicitShare indicates whether this permission was created explicitly
	// by a user to a grantee, or if it was implicitly created through the
	// creation of an object that inherited permissions of its parent
	ExplicitShare bool `db:"explicitShare"`
	// EncryptKey contains the encryption key for encrypting/decrypting the
	// content stream for this object at rest for this particular grantee and
	// revision
	EncryptKey []byte `db:"encryptKey"`
	// PermissionIV is a fresh random bitstring used for encrypting the key, and implicitly in the signature of the encrypt
	PermissionIV []byte `db:"permissionIV"`
	// PermissionMAC lets us authenticate that odrive actually wrote this permission
	PermissionMAC []byte `db:"permissionMAC"`
}

// ODObjectPermissionResultset encapsulates the ODObjectPermission defined
// herein as an array with resultset metric information to expose page size,
// page number, total rows, and page count information when retrieving from the
// database
type ODObjectPermissionResultset struct {
	Resultset
	Permissions []ODObjectPermission
}

/////These cannot be in crypto.go because of import cycles.  But they definitely make sense here

// CopyEncryptKey from one permission to another -- needs to be encapsulated because it's so easy to mess up
func CopyEncryptKey(passphrase string, src, dst *ODObjectPermission) {
	d := utils.ApplyPassphrase(passphrase, src.PermissionIV, src.EncryptKey)
	//Just reset the IV on every copy of a permission, to ensure that key and IV are consistent
	dst.PermissionIV = utils.CreatePermissionIV()
	dst.EncryptKey = utils.ApplyPassphrase(passphrase, dst.PermissionIV, d)
	dst.PermissionMAC = CalculatePermissionMAC(passphrase, dst)
}

// SetEncryptKey to create a permission that is not a copy of anything yet existing.
func SetEncryptKey(passphrase string, dst *ODObjectPermission) {
	k := utils.CreateKey()
	dst.PermissionIV = utils.CreatePermissionIV()
	dst.EncryptKey = utils.ApplyPassphrase(passphrase, dst.PermissionIV, k)
	dst.PermissionMAC = CalculatePermissionMAC(passphrase, dst)
}

// CalculatePermissionMAC - validate that odrive wrote this grant
func CalculatePermissionMAC(passphrase string, src *ODObjectPermission) []byte {
	//Sign our permission
	return utils.DoMAC(passphrase, src.PermissionIV, src.Grantee,
		src.AllowCreate,
		src.AllowRead,
		src.AllowUpdate,
		src.AllowDelete,
		src.AllowShare,
		src.EncryptKey,
	)
}

// EqualsPermissionMAC - check the integrity of this permission
func EqualsPermissionMAC(passphrase string, src *ODObjectPermission) bool {
	m := CalculatePermissionMAC(passphrase, src)
	lm := len(m)
	lp := len(src.PermissionMAC)
	if lm != lp {
		return false
	}
	for i := 0; i < len(m); i++ {
		if m[i] != src.PermissionMAC[i] {
			return false
		}
	}
	return true
}

package models

import (
	"fmt"
	"strings"

	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	configx "decipher.com/object-drive-server/configx"
)

// ODObjectPermission is a nestable structure defining the attributes for
// permissions granted on an object for users who have access to the object
// in Object Drive
type ODObjectPermission struct {
	ODCommonMeta
	ODChangeTracking
	// ObjectID identifies the object for which this permission applies.
	ObjectID []byte `db:"objectId"`
	// Grantee indicates the flattened representation of a user or group
	// referenced by a permission
	Grantee string `db:"grantee"`
	// AcmShare is used for inbound processing only composed of the share
	// for this grantee as either a user distinguished name value, or as a
	// project name, display name and group value defined in a json struct.
	// This value is built and captured for potential usage later to
	// reassemble a complete acm share structure composed of multiple
	// grantees.
	AcmShare   string `db:"acmShare"`
	AcmGrantee ODAcmGrantee
	ODCommonPermission
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

// IsReadOnly returns whether only the allowRead capability is being granted
func (permission *ODObjectPermission) IsReadOnly() bool {
	c := permission.AllowCreate
	r := permission.AllowRead
	u := permission.AllowUpdate
	d := permission.AllowDelete
	s := permission.AllowShare
	return !c && r && !u && !d && !s
}

// PermissionForUser is a helper function to create a new internal object permission object with a user distinguished name as the grantee
func PermissionForUser(user string, allowCreate bool, allowRead bool, allowUpdate bool, allowDelete bool, allowShare bool) ODObjectPermission {
	newPermission := ODObjectPermission{}
	newPermission.Grantee = AACFlatten(user)
	newPermission.AcmShare = fmt.Sprintf(`{"users":["%s"]}`, user)
	newPermission.AcmGrantee.Grantee = newPermission.Grantee
	newPermission.AcmGrantee.UserDistinguishedName = ToNullString(user)
	newPermission.AcmGrantee.DisplayName = ToNullString(configx.GetCommonName(user))
	newPermission.AllowCreate = allowCreate
	newPermission.AllowRead = allowRead
	newPermission.AllowUpdate = allowUpdate
	newPermission.AllowDelete = allowDelete
	newPermission.AllowShare = allowShare
	return newPermission
}

// PermissionForGroup is a helper function to create a new internal object permission object with a project name, display name and group name as the grantee
func PermissionForGroup(projectName string, projectDisplayName string, groupName string, allowCreate bool, allowRead bool, allowUpdate bool, allowDelete bool, allowShare bool) ODObjectPermission {
	newPermission := ODObjectPermission{}
	if len(strings.TrimSpace(projectDisplayName)) > 0 {
		newPermission.Grantee = AACFlatten(strings.TrimSpace(projectDisplayName + "_" + groupName))
	} else {
		newPermission.Grantee = AACFlatten(strings.TrimSpace(groupName))
	}
	// AAC seems to expect only one instance of case-insensitive key that projectName represents
	newPermission.AcmShare = fmt.Sprintf(`{"projects":{"%s":{"disp_nm":"%s","groups":["%s"]}}}`, strings.ToLower(projectName), projectDisplayName, groupName)
	newPermission.AcmGrantee.Grantee = newPermission.Grantee
	newPermission.AcmGrantee.ProjectName = ToNullString(strings.ToLower(projectName))
	newPermission.AcmGrantee.ProjectDisplayName = ToNullString(projectDisplayName)
	newPermission.AcmGrantee.GroupName = ToNullString(groupName)
	newPermission.AcmGrantee.DisplayName = ToNullString(strings.TrimSpace(projectDisplayName + " " + groupName))
	newPermission.AllowCreate = allowCreate
	newPermission.AllowRead = allowRead
	newPermission.AllowUpdate = allowUpdate
	newPermission.AllowDelete = allowDelete
	newPermission.AllowShare = allowShare
	return newPermission
}

// PermissionWithoutRead is a helper function that creates a new permission with the same settings as the permission passed in, without allowRead set.
func PermissionWithoutRead(i ODObjectPermission) ODObjectPermission {
	g := i.AcmGrantee
	if g.UserDistinguishedName.Valid {
		return PermissionForUser(g.UserDistinguishedName.String, i.AllowCreate, false, i.AllowUpdate, i.AllowDelete, i.AllowShare)
	}
	return PermissionForGroup(g.ProjectName.String, g.ProjectDisplayName.String, g.GroupName.String, i.AllowCreate, false, i.AllowUpdate, i.AllowDelete, i.AllowShare)
}

// copypasta from protocol
func AACFlatten(inVal string) string {
	emptyList := []string{" ", ",", "=", "'", ":", "(", ")", "$", "[", "]", "{", "}", "|", "\\"}
	underscoreList := []string{".", "-"}
	outVal := strings.ToLower(inVal)
	for _, s := range emptyList {
		outVal = strings.Replace(outVal, s, "", -1)
	}
	for _, s := range underscoreList {
		outVal = strings.Replace(outVal, s, "_", -1)
	}
	return outVal
}

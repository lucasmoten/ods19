package models

import (
	"fmt"
	"strings"
	"time"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/crypto"
)

// ODObjectPermission is a nestable structure defining the attributes for
// permissions granted on an object for users who have access to the object
// in Object Drive
type ODObjectPermission struct {
	// ID is the unique identifier for an item in Object Drive.
	ID []byte `db:"id"`
	// CreatedDate is the timestamp of when an item was created.
	CreatedDate time.Time `db:"createdDate"`
	// CreatedBy is the user, identified by distinguished name, that created this
	// item.
	CreatedBy string `db:"createdBy"`
	// ModifiedDate is the timestamp of when an item was modified. If an item
	// has only been created and not subsequently modified, its ModifiedDate
	// shall equate to the CreatedDate once stored in the repository.
	ModifiedDate time.Time `db:"modifiedDate"`
	// ModifiedBy is the user, identified by distinguished name, that last
	// modified this item
	ModifiedBy string `db:"modifiedBy"`
	// IsDeleted indicates whether the item is currently marked as deleted and
	// subsequently filtered from certain API results
	IsDeleted bool `db:"isDeleted" json:"-"`
	// DeletedDate is the timestamp of when an item was deleted, or null if it
	// currently is not deleted.
	DeletedDate NullTime `db:"deletedDate" json:"-"`
	// DeletedBy is the user, identified by distinguished name, that marked the
	// item as deleted, or null if the item is currently not deleted.
	DeletedBy NullString `db:"deletedBy" json:"-"`
	// ChangeCount indicates the number of times the item has been modified. For
	// newly created items, this value will reflect 0
	ChangeCount int `db:"changeCount"`
	// ChangeToken is generated value which is assigned at the database as a md5
	// hash of the concatencation of the id, changeCount, and most recent
	// modifiedDate as a string delimited by colons. For API calls performing
	// updates, the changeToken must be passed which will be compared against the
	// current value on the record. If properly implemented by callers, this will
	// prevent accidental overwrites.
	ChangeToken string `db:"changeToken"`
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

// CopyEncryptKey from one permission to another -- needs to be encapsulated because it's so easy to mess up
func CopyEncryptKey(passphrase string, src, dst *ODObjectPermission) {
	d := crypto.ApplyPassphrase(passphrase, src.PermissionIV, src.EncryptKey)
	//Just reset the IV on every copy of a permission, to ensure that key and IV are consistent
	dst.PermissionIV = crypto.CreatePermissionIV()
	dst.EncryptKey = crypto.ApplyPassphrase(passphrase, dst.PermissionIV, d)
	dst.PermissionMAC = CalculatePermissionMAC(passphrase, dst)
}

// SetEncryptKey to create a permission that is not a copy of anything yet existing.
func SetEncryptKey(passphrase string, dst *ODObjectPermission) {
	k := crypto.CreateKey()
	dst.PermissionIV = crypto.CreatePermissionIV()
	dst.EncryptKey = crypto.ApplyPassphrase(passphrase, dst.PermissionIV, k)
	dst.PermissionMAC = CalculatePermissionMAC(passphrase, dst)
}

// CalculatePermissionMAC - validate that odrive wrote this grant
func CalculatePermissionMAC(passphrase string, src *ODObjectPermission) []byte {
	//Sign our permission
	return crypto.DoMAC(passphrase, src.PermissionIV, src.Grantee,
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
	newPermission.AcmGrantee.DisplayName = ToNullString(config.GetCommonName(user))
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
	if g.UserDistinguishedName.Valid && len(g.UserDistinguishedName.String) > 0 {
		return PermissionForUser(g.UserDistinguishedName.String, i.AllowCreate, false, i.AllowUpdate, i.AllowDelete, i.AllowShare)
	}
	return PermissionForGroup(g.ProjectName.String, g.ProjectDisplayName.String, g.GroupName.String, i.AllowCreate, false, i.AllowUpdate, i.AllowDelete, i.AllowShare)
}

// PermissionForOwner creates permssion objects granting full CRUDS and read only access to designated owner as user or group
func PermissionForOwner(ownerResourceName string) (ownerCRUDS ODObjectPermission, ownerR ODObjectPermission) {
	odACMGrantee := NewODAcmGranteeFromResourceName(ownerResourceName)
	dn := odACMGrantee.UserDistinguishedName.String
	pn := odACMGrantee.ProjectName.String
	pdn := odACMGrantee.ProjectDisplayName.String
	gn := odACMGrantee.GroupName.String
	if len(odACMGrantee.UserDistinguishedName.String) > 0 {
		return PermissionForUser(dn, true, true, true, true, true), PermissionForUser(dn, false, true, false, false, false)
	}
	return PermissionForGroup(pn, pdn, gn, true, true, true, true, true), PermissionForGroup(pn, pdn, gn, false, true, false, false, false)
}

// AACFlatten is a copypasta from protocol
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

// IsCreating is a helper method to indicate if this permission is being created
func (permission *ODObjectPermission) IsCreating() bool {
	return (len(permission.ID) == 0)
}

func (permission ODObjectPermission) String() string {
	template := "[%s%s%s%s%s] %s %s"
	s := fmt.Sprintf(template,
		iifString(permission.AllowCreate, "C", "-"),
		iifString(permission.AllowRead, "R", "-"),
		iifString(permission.AllowUpdate, "U", "-"),
		iifString(permission.AllowDelete, "D", "-"),
		iifString(permission.AllowShare, "S", "-"),
		permission.Grantee,
		permission.AcmShare)
	return s
}

func getResourceNameFromAcmGrantee(acmGrantee ODAcmGrantee) string {
	return acmGrantee.ResourceName()
}

func getResourceNameFromAcmShare(acmShare string) string {
	if len(acmShare) > 0 {
		o := []string{}
		if strings.HasPrefix(acmShare, `{"users"`) {
			u := acmShare
			u = strings.Replace(u, `{"users":["`, "", 0)
			u = strings.Replace(u, `"]}`, "", 0)
			if len(u) > 0 {
				o = append(o, "user")
				o = append(o, u)
			}
		} else if strings.HasPrefix(acmShare, `{"projects"`) {
			groupName := acmShare
			groupName = strings.Replace(groupName, `{"projects":{"`, "", 0)
			projectName := groupName[0:strings.Index(groupName, `"`)]
			groupName = strings.Replace(groupName, projectName, "", 0)
			groupName = strings.Replace(groupName, `":{"disp_nm":"`, "", 0)
			projectDisplayName := groupName[0:strings.Index(groupName, `"`)]
			groupName = strings.Replace(groupName, projectDisplayName, "", 0)
			groupName = strings.Replace(groupName, `","groups":["`, "", 0)
			groupName = strings.Replace(groupName, `"]}}}`, "", 0)
			if len(projectName) > 0 && len(projectDisplayName) > 0 && len(groupName) > 0 {
				o = append(o, "group")
				o = append(o, projectName)
				o = append(o, projectDisplayName)
				o = append(o, groupName)
			} else if len(groupName) > 0 {
				o = append(o, "group")
				o = append(o, groupName)
			}
		}
		if len(o) > 0 {
			return strings.Join(o, "/")
		}
	}
	return ""
}

// GetResourceName returns a serialized resource name prefixed by type based upon
// the permissions acmGrantee or acmShare values.
func (permission ODObjectPermission) GetResourceName() string {
	if len(permission.AcmGrantee.DisplayName.String) > 0 {
		return getResourceNameFromAcmGrantee(permission.AcmGrantee)
	}
	if len(permission.AcmShare) > 0 {
		return getResourceNameFromAcmShare(permission.AcmShare)
	}
	return ""

}

func iifString(c bool, t string, f string) string {
	if c {
		return t
	}
	return f
}

// CreateODPermissionFromResource examines a resource string and prepares the basis of a permission from parsed values.
func CreateODPermissionFromResource(resource string) (ODObjectPermission, error) {
	if strings.HasPrefix(resource, "user/") {
		return createODPermissionFromUserResource(resource), nil
	}
	if strings.HasPrefix(resource, "group/") {
		return createODPermissionFromGroupResource(resource), nil
	}
	return ODObjectPermission{}, fmt.Errorf("Unhandled format for resource string")
}

func createODPermissionFromUserResource(resource string) ODObjectPermission {
	parts := strings.Split(strings.Replace(resource, "user/", "", 1), "/")
	return PermissionForUser(parts[0], false, false, false, false, false)
}

func createODPermissionFromGroupResource(resource string) ODObjectPermission {
	parts := strings.Split(strings.Replace(resource, "group/", "", 1), "/")
	switch len(parts) {
	case 1:
		return PermissionForGroup("", "", parts[0], false, false, false, false, false)
	case 2:
		return PermissionForGroup("", "", parts[1], false, false, false, false, false)
	default:
		return PermissionForGroup(parts[0], parts[1], parts[2], false, false, false, false, false)
	}
}

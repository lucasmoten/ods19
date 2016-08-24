package protocol

import (
	"fmt"

	"decipher.com/object-drive-server/metadata/models"
)

// String satisfies the Stringer interface for CallerPermission.
func (cp CallerPermission) String() string {
	template := "[create=%v read=%v update=%v delete=%v share=%v]"
	s := fmt.Sprintf(template, cp.AllowCreate, cp.AllowRead, cp.AllowUpdate, cp.AllowDelete, cp.AllowShare)
	return s
}

// WithRolledUp takes a slice of Permission and returns a new composite CallerPermission.
func (cp CallerPermission) WithRolledUp(caller Caller, perms ...Permission) CallerPermission {
	for _, perm := range perms {
		if perm.Grantee == models.AACFlatten(caller.DistinguishedName) || perm.Grantee == models.AACFlatten("-Everyone") || caller.InGroup(perm.Grantee) {
			cp.AllowCreate = cp.AllowCreate || perm.AllowCreate
			cp.AllowRead = cp.AllowRead || perm.AllowRead
			cp.AllowUpdate = cp.AllowUpdate || perm.AllowUpdate
			cp.AllowDelete = cp.AllowDelete || perm.AllowDelete
			cp.AllowShare = cp.AllowShare || perm.AllowShare
		}
	}
	return cp
}

package protocol

import (
	"fmt"
	"strings"
)

// CallerPermission is a structure defining the attributes for
// permissions granted on an object for the caller of an operation where an
// object is returned
type CallerPermission struct {
	// AllowCreate indicates whether the caller has permission to create child
	// objects beneath this object
	AllowCreate bool `json:"allowCreate"`
	// AllowRead indicates whether the caller has permission to read this
	// object. This is the most fundamental permission granted, and should always
	// be true as only records need to exist where permissions are granted as
	// the system denies access by default. Read access to an object is necessary
	// to perform any other action on the object.
	AllowRead bool `json:"allowRead"`
	// AllowUpdate indicates whether the caller has permission to update this
	// object
	AllowUpdate bool `json:"allowUpdate"`
	// AllowDelete indicates whether the caller has permission to delete this
	// object
	AllowDelete bool `json:"allowDelete"`
	// AllowShare indicates whether the caller has permission to view and
	// alter permissions on this object
	AllowShare bool `json:"allowShare"`
}

// String satisfies the Stringer interface for CallerPermission.
func (cp CallerPermission) String() string {
	template := "[create=%v read=%v update=%v delete=%v share=%v]"
	s := fmt.Sprintf(template, cp.AllowCreate, cp.AllowRead, cp.AllowUpdate, cp.AllowDelete, cp.AllowShare)
	return s
}

// WithRolledUp takes a slice of Permission and returns a new composite CallerPermission.
func (cp CallerPermission) WithRolledUp(caller Caller, perms ...Permission) CallerPermission {
	for _, perm := range perms {
		if perm.Grantee == flatten(caller.DistinguishedName) || perm.Grantee == "-Everyone" || caller.InGroup(perm.Grantee) {
			cp.AllowCreate = cp.AllowCreate || perm.AllowCreate
			cp.AllowRead = cp.AllowRead || perm.AllowRead
			cp.AllowUpdate = cp.AllowUpdate || perm.AllowUpdate
			cp.AllowDelete = cp.AllowDelete || perm.AllowDelete
			cp.AllowShare = cp.AllowShare || perm.AllowShare
		}
	}
	return cp
}

func flatten(inVal string) string {
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

package protocol

import (
	"fmt"
	"time"
)

// Permission is a nestable structure defining the attributes for
// permissions granted on an object for users who have access to the object
// in Object Drive
type Permission struct {
	// ID is the unique identifier for this permission in Object Drive.
	ID string `json:"-"`
	// CreatedDate is the timestamp of when a permission was created.
	CreatedDate time.Time `json:"-"`
	// CreatedBy is the user that created this permission.
	CreatedBy string `json:"-"`
	// ModifiedDate is the timestamp of when a permission was modified or created.
	ModifiedDate time.Time `json:"-"`
	// ModifiedBy is the user that last modified this permission
	ModifiedBy string `json:"-"`
	// ChangeCount indicates the number of times the permission has been modified.
	ChangeCount int `json:"-"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"-"`
	// ObjectID identifies the object for which this permission applies.
	ObjectID string `json:"-"`
	// Grantee indicates the flattened representation of a user or group
	// referenced by a permission
	Grantee string `json:"grantee,omitempty"`
	// ProjectName contains the project key portion of an AcmShare if this
	// grantee represents a group
	ProjectName string `json:"projectName,omitempty"`
	// ProjectDisplayName contains the disp_nm portion of an AcmShare if this
	// grantee represents a group
	ProjectDisplayName string `json:"projectDisplayName,omitempty"`
	// GroupName contains the group value portion of an AcmShare if this
	// grantee represents a group
	GroupName string `json:"groupName,omitempty"`
	// UserDistinguishedName contains a user value portion of an AcmShare
	// if this grantee represnts a user
	UserDistinguishedName string `json:"userDistinguishedName,omitempty"`
	// DisplayName is a friendly display name suitable for user interfaces for
	// the grantee modeleed on the distinguished name common name, or project and group
	DisplayName string `json:"displayName,omitempty"`
	// AllowCreate indicates whether the grantee has permission to create child
	// objects beneath this object
	AllowCreate bool `json:"allowCreate"`
	// AllowRead indicates whether the grantee has permission to read this
	// object. This is the most fundamental permission granted, and should always
	// be true as only records need to exist where permissions are granted as
	// the system denies access by default. Read access to an object is necessary
	// to perform any other action on the object.
	AllowRead bool `json:"allowRead"`
	// AllowUpdate indicates whether the grantee has permission to update this
	// object
	AllowUpdate bool `json:"allowUpdate"`
	// AllowDelete indicates whether the grantee has permission to delete this
	// object
	AllowDelete bool `json:"allowDelete"`
	// AllowShare indicates whether the grantee has permission to view and
	// alter permissions on this object
	AllowShare bool `json:"allowShare"`
	// ExplicitShare indicates whether this permission was created explicitly
	// by a user to a grantee, or if it was implicitly created through the
	// creation of an object that inherited permissions of its parent
	ExplicitShare bool `json:"-"`
}

// CreatePermission is a nestable structure defining the attributes for
// permissions to be created on an object when an object is created.
type CreatePermission struct {
	// ProjectName contains the project key portion of an AcmShare if this
	// grantee represents a group
	ProjectName string `json:"projectName,omitempty"`
	// ProjectDisplayName contains the disp_nm portion of an AcmShare if this
	// grantee represents a group
	ProjectDisplayName string `json:"projectDisplayName,omitempty"`
	// GroupName contains the group value portion of an AcmShare if this
	// grantee represents a group
	GroupName string `json:"groupName,omitempty"`
	// UserDistinguishedName contains a user value portion of an AcmShare
	// if this grantee represnts a user
	UserDistinguishedName string `json:"userDistinguishedName,omitempty"`
	// AllowCreate indicates whether the grantee has permission to create child
	// objects beneath this object
	AllowCreate bool `json:"allowCreate"`
	// AllowRead indicates whether the grantee has permission to read this
	// object. This is the most fundamental permission granted, and should always
	// be true as only records need to exist where permissions are granted as
	// the system denies access by default. Read access to an object is necessary
	// to perform any other action on the object.
	AllowRead bool `json:"allowRead"`
	// AllowUpdate indicates whether the grantee has permission to update this
	// object
	AllowUpdate bool `json:"allowUpdate"`
	// AllowDelete indicates whether the grantee has permission to delete this
	// object
	AllowDelete bool `json:"allowDelete"`
	// AllowShare indicates whether the grantee has permission to view and
	// alter permissions on this object
	AllowShare bool `json:"allowShare"`
}

// String satisfies the Stringer interface for Permission.
func (p Permission) String() string {
	// template := "[create=%v read=%v update=%v delete=%v share=%v grantee=%v]"
	// s := fmt.Sprintf(template, p.AllowCreate, p.AllowRead, p.AllowUpdate, p.AllowDelete, p.AllowShare, p.Grantee)
	template := "[%s%s%s%s%s] %s"
	s := fmt.Sprintf(template,
		iifString(p.AllowCreate, "C", "-"),
		iifString(p.AllowRead, "R", "-"),
		iifString(p.AllowUpdate, "U", "-"),
		iifString(p.AllowDelete, "D", "-"),
		iifString(p.AllowShare, "S", "-"),
		p.DisplayName)
	return s
}
func iifString(c bool, t string, f string) string {
	if c {
		return t
	}
	return f
}

package protocol

import "time"

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
	// Grantee indicates the user, identified by distinguishedName from the user
	// table for which this grant applies
	Grantee string `json:"grantee,omitempty"`
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

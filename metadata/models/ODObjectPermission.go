package models

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
	// EncryptKey contains the encryption key for encrypting/decrypting the
	// content stream for this object at rest for this particular grantee and
	// revision
	EncryptKey []byte `db:"encryptKey"`
}

// ODObjectPermissionResultset encapsulates the ODObjectPermission defined
// herein as an array with resultset metric information to expose page size,
// page number, total rows, and page count information when retrieving from the
// database
type ODObjectPermissionResultset struct {
	Resultset
	Permissions []ODObjectPermission
}

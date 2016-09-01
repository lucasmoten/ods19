package protocol

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

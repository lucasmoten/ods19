package protocol

// GroupSpace is a nestable structure defining the base attributes for a GroupSpace
type GroupSpace struct {
	// Grantee indicates the flattened representation of a user or group
	Grantee string `json:"grantee"`
	// ResourceString is the built up resource name as stored in the database
	ResourceString string `json:"resourceString"`
	// DisplayName is a UI friendly representation of the resource string
	DisplayName string `json:"displayName"`
	// Quantity indicates how many root objects found for this group in user context
	Quantity int `json:"quantity"`
}

// GroupSpaceResultset encapsulates the GroupSpace defined herein as an array with
// resultset metric information to expose page size, page number, total rows,
// and page count information when retrieving from the database
type GroupSpaceResultset struct {
	Resultset
	GroupSpaces []GroupSpace `json:"groups,omitempty"`
}

package protocol

// UpdateObjectAndStreamRequest is a subset of Object for use to disallow providing certain fields.
type UpdateObjectAndStreamRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// TypeID references the ODObjectType by its ID indicating the type of this
	// object
	TypeID string `json:"typeId,omitempty"`
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
	// Description is an abstract of the object or its contents
	Description string `json:"description"`
	// RawACM is the raw ACM string that got supplied to modify this object
	RawAcm interface{} `json:"acm"`
	// Permission is the API 1.1+ version for providing permissions for users and groups with a resource and capability driven approach
	Permission Permission `json:"permission,omitempty"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `json:"contentType,omitempty"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize int64 `json:"contentSize,omitempty"`
	// ContainsUSPersonsData indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData,omitEmpty"`
	// ExemptFromFOIA indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA,omitEmpty"`
	// Properties is an array of Object Properties associated with this object
	Properties []Property `json:"properties,omitempty"`
	// RecursiveShare, if true, will apply the updated share permissions to all child objects.
	RecursiveShare bool `json:"recursiveShare"`
}

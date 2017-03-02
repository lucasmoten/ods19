package protocol

// CreateObjectRequest is a subset of Object for use to disallow providing certain fields.
type CreateObjectRequest struct {
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
	// NamePathDelimiter is an optional delimiter for which the name value is intended
	// to be broken up to create intermediate objects to represent a hierarchy.  by
	// default this value is internally processed as the record separator (char 30).
	// It may be overridden by providing a different string here
	NamePathDelimiter string `json:"namePathDelimiter,omitempty"`
	// Description is an abstract of the object or its contents
	Description string `json:"description"`
	// ParentID can optionally reference another object by its ID in hexadecimal string
	// format to denote the parent/ancestor of this object for hierarchical purposes.
	// Leaving this field empty, or not present indicates that the object should be
	// created at the 'root' or 'top level'
	ParentID string `json:"parentId,omitempty"`
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm interface{} `json:"acm"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `json:"contentType,omitempty"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize int64 `json:"contentSize,omitempty"`
	// Permission is the API 1.1+ version for providing permissions for users and groups with a resource and capability driven approach
	Permission Permission `json:"permission,omitempty"`
	// ContainsUSPersonsData indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData,omitEmpty"`
	// ExemptFromFOIA indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA,omitEmpty"`
	// Properties is an array of Object Properties associated with this object
	Properties []Property `json:"properties,omitempty"`
	// Permissions is the API 1.0 version for providing permissions for users and groups with a share model
	Permissions []ObjectShare `json:"permissions,omitempty"`
}

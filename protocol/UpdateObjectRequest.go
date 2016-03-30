package protocol

// UpdateObjectRequest is a subset of Object for use to disallow providing certain fields.
type UpdateObjectRequest struct {
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
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm string `json:"acm"`
	// Properties is an array of Object Properties associated with this object
	Properties []Property `json:"properties,omitempty"`
}

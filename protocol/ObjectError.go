package protocol

// ObjectError is a simple list of object identifiers
type ObjectError struct {
	// ObjectID is the unique identifier for an object
	ObjectID string `json:"objectId,omitempty"`
	// Error is an error string for the operation being performed on this object
	Error string `json:"error,omitempty"`
	// Msg is an informative message about the error that transpired
	Msg string `json:"msg,omitempty"`
	// Code is an status code associated with the operation for this identifier
	Code int `json:"code,omitempty"`
}

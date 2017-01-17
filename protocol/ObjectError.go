package protocol

// ObjectError is a simple list of object identifiers
type ObjectError struct {
	ObjectID string `json:"objectId,omitempty"`
	Error    string `json:"error,omitempty"`
	Msg      string `json:"msg,omitempty"`
	Code     int    `json:"code,omitempty"`
}

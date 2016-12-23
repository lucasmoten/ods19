package protocol

// ObjectError is a simple list of object identifiers
type ObjectError struct {
	ObjectID string `json:"objectId"`
	Error    string `json:"error"`
	Msg      string `json:"msg"`
	Code     int    `json:"code"`
}

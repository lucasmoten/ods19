package protocol

import "time"

// DeletedObjectResponse is the response information provided when an object
// is deleted from Object Drive
type DeletedObjectResponse struct {
	ID string
	// DeletedDate is the timestamp of when an item was deleted.
	DeletedDate time.Time `json:"deletedDate"`
	// CallerPermission is the composite permission the caller has for this object
	CallerPermission CallerPermission `json:"callerPermission,omitempty"`
}

// WithCallerPermission rolls up permissions for a caller, sets them on a copy of
// the DeletedObjectResponse, and returns that copy.
func (obj DeletedObjectResponse) WithCallerPermission(caller Caller) DeletedObjectResponse {
	obj.CallerPermission = CallerPermission{AllowRead: true, AllowDelete: true}
	return obj
}

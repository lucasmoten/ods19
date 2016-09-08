package protocol

import "time"

// ExpungedObjectResponse is the response information provided when an object
// is expunged from Object Drive
type ExpungedObjectResponse struct {
	// ExpungedDate is the timestamp of when an item was deleted permanently.
	ExpungedDate time.Time `json:"expungedDate"`
	// CallerPermission is the composite permission the caller has for this object
	CallerPermission CallerPermission `json:"callerPermission,omitempty"`
}

// WithCallerPermission rolls up permissions for a caller, sets them on a copy of
// the ExpungedObjectResponse, and returns that copy.
func (obj ExpungedObjectResponse) WithCallerPermission(caller Caller) ExpungedObjectResponse {
	obj.CallerPermission = CallerPermission{AllowRead: true, AllowDelete: true}
	return obj
}

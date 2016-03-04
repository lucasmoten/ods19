package protocol

import "time"

// ExpungedObjectResponse is the response information provided when an object
// is expunged from Object Drive
type ExpungedObjectResponse struct {
	// ExpungedDate is the timestamp of when an item was deleted permanently.
	ExpungedDate time.Time `json:"expungedDate"`
}

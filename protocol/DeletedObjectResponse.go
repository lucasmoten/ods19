package protocol

import "time"

// DeletedObjectResponse is the response information provided when an object
// is deleted from Object Drive
type DeletedObjectResponse struct {
	ID string
	// DeletedDate is the timestamp of when an item was deleted.
	DeletedDate time.Time `json:"deletedDate"`
}

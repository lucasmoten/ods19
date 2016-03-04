package protocol

import "time"

// RemovedObjectShareResponse is the response information provided when an
// object share is deleted from Object Drive
type RemovedObjectShareResponse struct {
	// DeletedDate is the timestamp of when an object share was deleted.
	DeletedDate time.Time `json:"deletedDate"`
}

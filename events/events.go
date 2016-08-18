package events

import "encoding/json"

// Event defines a type that can yield itself as JSON bytes.
type Event interface {
	Yield() []byte
}

// Index defines an Event suitable for Finder search indexing service.
type Index struct {
	// Action describes the object event being published. One of create, read,
	// update, delete, undelete.
	Action string `json:"action"`
	// ObjectID is a 32 character hex encoded string corresponding to the database ID.
	ObjectID string `json:"object_id"`
	// Timestamp is an RFC3339 timestamp generated when the event was created.
	Timestamp string `json:"timestamp"`
	// ChangeToken is a random string regenerated with each update to an object.
	// A successful update to an object must present the most-current ChangeToken.
	ChangeToken string `json:"change_token"`
	// StreamUpdate indicates whether an event corresponds to a server action
	// that changed the bytes of the file.
	StreamUpdate bool `json:"stream_update"`
	// UserDN identifies the user that triggered the action.
	UserDN string `json:"user_dn"`
	// SessionID is a random string generated for each http request.
	SessionID string `json:"session_id"`
}

// Yield satisfies the Event interface.
func (i Index) Yield() []byte {
	b, _ := json.Marshal(i)
	return b
}

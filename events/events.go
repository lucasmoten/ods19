package events

import "encoding/json"

// Event defines a type that can yield itself as JSON bytes.
type Event interface {
	Yield() []byte
}

// Index defines an Event suitable for Finder search indexing service.
type Index struct {
	Action       string `json:"action"`
	ObjectID     string `json:"object_id"`
	Timestamp    string `json:"timestamp"`
	ChangeToken  string `json:"change_token"`
	StreamUpdate bool   `json:"stream_update"`
	UserDN       string `json:"user_dn"`
	SessionID    string `json:"session_id"`
}

// Yield satisfies the Event interface.
func (i Index) Yield() []byte {
	b, _ := json.Marshal(i)
	return b
}

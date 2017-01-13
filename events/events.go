package events

import (
	"encoding/json"

	auditevent "github.com/deciphernow/gm-fabric-go/audit/events_thrift"
)

// Event defines a type that can yield itself as JSON bytes.
type Event interface {
	Yield() []byte
}

// GEM stands for Global Event Model, the standard envelope for events that all
// microservices must publish. This type is manually maintained for now, but
// may rely on code generation in the future.
type GEM struct {
	// A software-generated GUID, unique for every published event
	ID string `json:"eventId"`
	// An array of GUIDs. Will be empty if the event is never "enriched"
	EventChain []string `json:"eventChain"`
	// String for major and minor version of the event schema, e.g. "1.0"
	SchemaVersion string `json:"schemaVersion"`
	// Identifiers, usually DNs, for the originators of the event (end user
	// and/or system users and/or system impersonators)
	OriginatorTokens []string `json:"originatorToken"`
	// A globally unique string identifying the source system, e.g. "object-drive-event"
	EventType string `json:"eventType"`
	// Unix timestamp, a numeric type in JSON
	Timestamp int64 `json:"timestamp"`
	// The IP address of the end user.
	XForwardedForIP string `json:"xForwardedForIp"`
	// The IP address of the system that emitted the event
	SystemIP string `json:"systemIp"`
	// A string identifying the action. One of: create, read,
	// update, delete, undelete.
	Action string `json:"action"`
	// Payload is the app-specific event we must provide.
	Payload ObjectDriveEvent `json:"payload"`
}

// Yield satisfies the Event interface.
func (e GEM) Yield() []byte {
	b, _ := json.Marshal(e)
	return b
}

// ObjectDriveEvent defines an Event suitable for many system subscribers. Fields
// may need to be added as the needs of subscribers change. The goal is to publish
// as single event stream that supports auditing, indexing, and more. Note that
// this type is to be embedded in the Global Event Model (GEM).
type ObjectDriveEvent struct {
	// Audit embeds the ICS 500-27 schema
	Audit auditevent.AuditEvent `json:"audit_event"`
	// ObjectID is a 32 character hex encoded string corresponding to the database ID.
	ObjectID string `json:"object_id"`
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
func (e ObjectDriveEvent) Yield() []byte {
	b, _ := json.Marshal(e)
	return b
}

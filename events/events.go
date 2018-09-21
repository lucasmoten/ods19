package events

import (
	"encoding/json"

	"bitbucket.di2e.net/dime/object-drive-server/protocol"

	auditevent "bitbucket.di2e.net/greymatter/gov-go/audit/events_thrift"
)

// Event defines a type that can yield itself as JSON bytes.
type Event interface {
	Yield() []byte
	EventAction() string
	IsSuccessful() bool
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

// EventAction satisfies the Event interface
func (e GEM) EventAction() string {
	return e.Action
}

// IsSuccessful satisfies the Event interface
func (e GEM) IsSuccessful() bool {
	if e.Payload.Audit.ActionResult == nil {
		return false
	}
	eventResult := *e.Payload.Audit.ActionResult
	return eventResult == "SUCCESS"
}

// ObjectDriveEvent defines an Event suitable for many system subscribers. Fields
// may need to be added as the needs of subscribers change. The goal is to publish
// as single event stream that supports auditing, indexing, and more. Note that
// this type is to be embedded in the Global Event Model (GEM).
type ObjectDriveEvent struct {
	// Audit (since 1.0) embeds the ICS 500-27 schema
	Audit auditevent.AuditEvent `json:"audit_event"`
	// ObjectID (since 1.0) is a 32 character hex encoded string corresponding to the database ID.
	ObjectID string `json:"object_id"`
	// ChangeToken (since 1.0) is generated value assigned as a hash based upon the id, change count, and modified date
	ChangeToken string `json:"change_token"`
	// StreamUpdate (since 1.0) indicates whether an event corresponds to a server action
	// that changed the bytes of the file.
	StreamUpdate bool `json:"stream_update"`
	// UserDN (since 1.0) identifies the user that triggered the action.
	UserDN string `json:"user_dn"`
	// SessionID (since 1.0) is a random string generated for each http request.
	SessionID string `json:"session_id"`
	// ---------------------------------------------------------------
	// CreatedBy (since 1.0.12) is the user that created this item.
	CreatedBy string `json:"createdBy,omitempty"`
	// ModifiedBy (since 1.0.12) is the user that last modified this item
	ModifiedBy string `json:"modifiedBy,omitempty"`
	// DeletedBy (since 1.0.12) is the user that last modified this item
	DeletedBy string `json:"deletedBy,omitempty"`
	// ChangeCount (since 1.0.12) indicates the number of times the item has been modified.
	ChangeCount int `json:"changeCount,omitempty"`
	// OwnedBy (since 1.0.12) indicates the individual user or group owning the object with full permissions
	OwnedBy string `json:"ownedBy,omitempty"`
	// ObjectType (since 1.0.12) reflects the name of the object type associated with TypeID
	ObjectType string `json:"objectType,omitempty"`
	// Name (since 1.0.12) is the given name for the object. (e.g., filename)
	Name string `json:"name,omitempty"`
	// Description (since 1.0.12) is an abstract of the object or its contents
	Description string `json:"description,omitempty"`
	// ParentID (since 1.0.12) refers to another object by id that is the parent of the one referenced herein
	ParentID string `json:"parentId,omitempty"`
	// ContentType (since 1.0.12) indicates the mime-type for the object contents
	ContentType string `json:"contentType,omitempty"`
	// ContentSize (since 1.0.12) denotes the length of the content stream for this object in bytes
	ContentSize int64 `json:"contentSize,omitempty"`
	// ContentHash (since 1.0.12) is a sha256 hash of the plaintext as hex encoded string
	ContentHash string `json:"contentHash,omitempty"`
	// ContainsUSPersonsData (since 1.0.12) indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData,omitempty"`
	// ExemptFromFOIA (since 1.0.12) indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA,omitempty"`
	// Breadcrumbs (since 1.0.12) is an array of Breadcrumb that may be returned on some API calls.
	// Clients can use breadcrumbs to display a list of parents. The top-level
	// parent should be the first item in the slice.
	Breadcrumbs []protocol.Breadcrumb `json:"breadcrumbs,omitempty"`
}

// Yield satisfies the Event interface.
func (e ObjectDriveEvent) Yield() []byte {
	b, _ := json.Marshal(e)
	return b
}

// WithEnrichedPayload populates payload fields with information from a protocol object
func WithEnrichedPayload(p ObjectDriveEvent, i protocol.Object) ObjectDriveEvent {
	p.Breadcrumbs = i.Breadcrumbs
	p.ChangeCount = i.ChangeCount
	p.ChangeToken = i.ChangeToken
	p.ContainsUSPersonsData = i.ContainsUSPersonsData
	p.ContentHash = i.ContentHash
	p.ContentSize = i.ContentSize
	p.ContentType = i.ContentType
	p.CreatedBy = i.CreatedBy
	p.DeletedBy = i.DeletedBy
	p.Description = i.Description
	p.ExemptFromFOIA = i.ExemptFromFOIA
	p.ModifiedBy = i.ModifiedBy
	p.Name = i.Name
	p.ObjectID = i.ID
	p.ObjectType = i.TypeName
	p.OwnedBy = i.OwnedBy
	p.ParentID = i.ParentID
	return p
}

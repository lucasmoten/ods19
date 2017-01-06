package server

import (
	"net/http"
	"time"

	"github.com/deciphernow/gm-fabric-go/audit/events_thrift"

	"decipher.com/object-drive-server/events"
)

// globalEventFromRequest sets up a standard set of fields on the global event model.
// Invoke this function before a specific route is chosen.
func globalEventFromRequest(r *http.Request) events.GEM {
	e := events.GEM{
		ID:            newGUID(),
		SchemaVersion: "1.0",
		OriginatorTokens: []string{
			r.Header.Get("EXTERNAL_SYS_DN"),
			r.Header.Get("USER_DN"),
		},
		EventType:       "object-drive-event",
		SystemIP:        resolveOurIP(),
		XForwardedForIP: r.Header.Get("X-Forwarded-For"),
		Timestamp:       time.Now().Unix(),
	}
	return e
}

// TODO put audit default here?
func defaultAudit(r *http.Request) events_thrift.AuditEvent {

	stringPtr := func(s string) *string { return &s }
	_ = stringPtr
	// Set a string pointer field on Audit like this
	// e.CreatedOn = stringPtr(fmt.Sprintf("%s", time.Now().Format(time.RFC3339)))

	var e events_thrift.AuditEvent
	return e
}

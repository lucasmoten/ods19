package server

import (
	"net/http"
	"time"

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

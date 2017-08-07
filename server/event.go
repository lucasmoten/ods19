package server

import (
	"net/http"
	"time"

	"github.com/deciphernow/gov-go/audit/events_thrift"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/util"
)

// globalEventFromRequest extracts data from the request and sets up a
// standard set of fields on the global event model.
func globalEventFromRequest(r *http.Request) events.GEM {
	e := events.GEM{
		ID:              newGUID(),
		SchemaVersion:   "1.0",
		EventType:       "object-drive-event",
		SystemIP:        util.GetIP(config.RootLogger),
		XForwardedForIP: r.Header.Get("X-Forwarded-For"),
		Timestamp:       time.Now().Unix(),
		Action:          "unknown",
	}
	if len(r.Header.Get("EXTERNAL_SYS_DN")) > 0 {
		e.OriginatorTokens = append(e.OriginatorTokens, r.Header.Get("EXTERNAL_SYS_DN"))
	}
	if len(r.Header.Get("USER_DN")) > 0 {
		e.OriginatorTokens = append(e.OriginatorTokens, r.Header.Get("USER_DN"))
	}

	return e
}

func defaultAudit(r *http.Request) events_thrift.AuditEvent {

	var e events_thrift.AuditEvent
	fqdn := r.URL.Host
	if len(fqdn) == 0 {
		fqdn = r.Host
	}
	e = audit.WithActionTargetWithoutAcm(e, "FULLY_QUALIFIED_DOMAIN_NAME", fqdn)
	if len(r.URL.String()) > 0 {
		e = audit.WithAdditionalInfo(e, "URL", r.URL.String())
	}
	if len(r.Header.Get("EXTERNAL_SYS_DN")) > 0 {
		e = audit.WithAdditionalInfo(e, "EXTERNAL_SYS_DN", r.Header.Get("EXTERNAL_SYS_DN"))
	}
	if len(r.Header.Get("USER_DN")) > 0 {
		e = audit.WithAdditionalInfo(e, "USER_DN", r.Header.Get("USER_DN"))
	}
	if len(r.Header.Get("SSL_CLIENT_S_DN")) > 0 {
		e = audit.WithAdditionalInfo(e, "SSL_CLIENT_S_DN", r.Header.Get("SSL_CLIENT_S_DN"))
	}
	e = audit.WithActionTargetVersions(e, "1.0")
	query := r.URL.RawQuery
	if len(query) > 0 {
		e = audit.WithQueryString(e, r.URL.RawQuery)
	}
	e = audit.WithType(e, "EventUnknown")
	e = audit.WithAction(e, "ACCESS")
	e = audit.WithActionMode(e, "USER_INITIATED")
	e = audit.WithActionResult(e, "FAILURE")
	e = audit.WithActionInitiator(e, "DISTINGUISHED_NAME", config.GetNormalizedDistinguishedName(r.Header.Get("USER_DN")))
	application := r.Header.Get("APPLICATION")
	if len(application) == 0 {
		application = "Object Drive"
	}
	e = audit.WithCreator(e, "APPLICATION", application)
	e = audit.WithCreatedOn(e, time.Now().UTC().Format("2006-01-02T15:04:05.000Z"))

	return e
}

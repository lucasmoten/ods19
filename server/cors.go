package server

import (
	"net/http"

	"bitbucket.di2e.net/dime/object-drive-server/services/audit"

	"fmt"

	"golang.org/x/net/context"
)

// cors requests handled here. This is a fully permissive cors implementation that instructs the
// client web browser to remove security restrictions regarding cross origin requests.
// see: http://enable-cors.org/server_nginx.html
func (h AppServer) cors(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	if r.Header.Get("Origin") == "" {
		herr := NewAppError(http.StatusBadRequest, fmt.Errorf("Origin must be specificed in CORS Preflight request"), "missing origin")
		h.publishError(gem, herr)
		return herr
	}

	// Reflect back headers as permissive
	//
	// If a UI front-end references this API, and also hosts malware javascript from a different
	// domain, then the other domain can make requests and perform operations on the user's
	// behalf.
	reqM := "GET, PUT, DELETE, POST, HEAD, OPTIONS"
	reqH := r.Header.Get("Access-Control-Request-Headers")
	if reqH == "" {
		reqH = "content-type, x-requested-with"
	}
	w.Header().Set("Access-Control-Allow-Methods", reqM)
	w.Header().Set("Access-Control-Allow-Headers", reqH)
	w.Header().Set("Access-Control-Max-Age", "600")
	w.Header().Set("Content-Type", "text/plain charset=UTF-8")
	w.Header().Set("Content-Length", "0")

	herr := NewAppError(http.StatusNoContent, nil, "No content")
	h.publishSuccess(gem, w)
	return herr
}

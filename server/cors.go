package server

import (
	"net/http"

	"fmt"

	"golang.org/x/net/context"
)

// cors requests handled here
// see: http://enable-cors.org/server_nginx.html
func (h AppServer) cors(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	if r.Header.Get("Origin") == "" {
		return NewAppError(400, fmt.Errorf("Origin must be specificed in CORS Preflight request"), "missing origin")
	}

	//
	// The basic idea is to reflect back headers.  "Would you accept $X?".  "We would accept $X."
	// The important part was already done in the ServeHTTP method
	// where Origin (which might better be called Access-Control-Request-Origin) was reflected
	// back as Access-Control-Request-Oritin -- not as '*'.  The point of this is that if a UI front-end
	// references the odrive API, and also hosts a malware ad-banner in the same page,
	// the malware ad-banner site should not get access to objects that the UI got from odrive.
	//

	reqM := "GET, PUT, DELETE, POST, HEAD, OPTIONS"
	reqH := r.Header.Get("Access-Control-Request-Headers")
	if reqH == "" {
		reqH = "content-type, x-requested-with"
	}

	// Set these headers.  We don't play security through obscurity with our API, so there is no reason
	// to nitpick on restrictions, which are enforced through better mechanisms such as
	// the 2way ssl authentication.  URLs are bound to a particular method by design, and wont get confused
	// by unexpected headers.

	w.Header().Set("Access-Control-Allow-Methods", reqM)
	w.Header().Set("Access-Control-Allow-Headers", reqH)
	w.Header().Set("Access-Control-Max-Age", "600")
	w.Header().Set("Content-Type", "text/plain charset=UTF-8")
	w.Header().Set("Content-Length", "0")
	return NewAppError(204, nil, "preflight")
	//return nil
}

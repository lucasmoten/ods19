package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) removeObjectSubscription(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "removeObjectSubscription", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

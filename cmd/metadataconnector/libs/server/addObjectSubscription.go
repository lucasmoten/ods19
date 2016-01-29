package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) addObjectSubscription(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "addObjectSubscription", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

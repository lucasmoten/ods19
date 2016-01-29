package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) getObjectStreamForRevision(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "getObjectStreamForRevision", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

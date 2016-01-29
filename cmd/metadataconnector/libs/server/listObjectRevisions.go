package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) listObjectRevisions(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "listObjectRevisions", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

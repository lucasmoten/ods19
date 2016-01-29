package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) updateObjectPermissions(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "updateObjectPermissions", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

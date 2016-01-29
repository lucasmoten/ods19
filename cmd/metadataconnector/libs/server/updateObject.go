package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) updateObject(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "updateObject", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

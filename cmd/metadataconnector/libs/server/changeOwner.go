package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) changeOwner(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "changeOwner", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) moveObject(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "moveObject", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

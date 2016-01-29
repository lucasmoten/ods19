package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) addObjectToFolder(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "addObjectToFolder", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

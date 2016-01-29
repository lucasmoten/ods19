package server

import (
	"fmt"
	"net/http"
)

func (h AppServer) listObjectsImages(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "listObjectsImages", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

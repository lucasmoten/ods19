package server

import (
	"fmt"
	"net/http"

	cfg "decipher.com/oduploader/config"
)

func (h AppServer) listObjectsSubscriptions(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, cfg.RootURL, "listObjectsSubscriptions", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

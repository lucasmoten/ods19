package server

import (
	"fmt"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
)

func (h AppServer) listUserObjectsShared(w http.ResponseWriter, r *http.Request) {
	who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "listUserObjectsShared", who)
	fmt.Fprintf(w, pageTemplateEnd)
}
package server

import (
	"fmt"
	"net/http"
	"strconv"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
)

// home is a method handler on AppServer for displaying a response when the
// root URI is requested without an operation. In this context, a UI is provided
// listing and linking to some available operations
func (h AppServer) home(w http.ResponseWriter, r *http.Request) {
	who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "Home Page", who)
	fmt.Fprintf(w, "Length of distinguished name: "+strconv.Itoa(len(who)))
	rootURL := "/service/metadataconnector/1.0"
	fmt.Fprintf(w, `
<hr/>
<h1>Microservice API</h1>

<a href="%s/object">Create Object</a> - Normally, this operation is a POST
	only to ../object-drive/object to add a new object to the system. When you
	click this link, a form will be displayed allowing you to set the name and
	type, and specify a file

<p />

<a href="%s/objects">List Objects</a> - This operation will result in a GET
	call to list root objects with default paging.

		`, rootURL, rootURL)
	fmt.Fprintf(w, pageTemplateEnd)
}

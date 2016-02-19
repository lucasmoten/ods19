package server

import "net/http"

// home is a method handler on AppServer for displaying a response when the
// root URI is requested without an operation. In this context, a UI is provided
// listing and linking to some available operations
func (h AppServer) home(w http.ResponseWriter, r *http.Request, caller Caller) {

	tmpl := h.TemplateCache.Lookup("home.html")

	// Anonymous struct syntax is tricky.
	apiFuncs := []struct{ Name, RelativeLink, Description string }{
		{"List Objects", "/service/metadataconnector/1.0/home/listObjects", "This operation will result in a GET call to list root objects with default paging."},
		{"Statistics", "/service/metadataconnector/1.0/stats", "This operation will result in a GET call to list root objects with default paging."},
		{"Users", "/service/metadataconnector/1.0/users", "This is a list of all users."},
	}

	data := struct {
		DistinguishedName string
		APIFunctions      []struct{ Name, RelativeLink, Description string }
	}{
		DistinguishedName: caller.DistinguishedName,
		APIFunctions:      apiFuncs,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

package server

import (
	"errors"
	"net/http"

	cfg "decipher.com/oduploader/config"
	"golang.org/x/net/context"
)

// home is a method handler on AppServer for displaying a response when the
// root URI is requested without an operation. In this context, a UI is provided
// listing and linking to some available operations
func (h AppServer) home(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	tmpl := h.TemplateCache.Lookup("home.html")

	// Anonymous struct syntax is tricky.
	apiFuncs := []struct{ Name, RelativeLink, Description string }{
		{"List Objects", cfg.RootURL + "/home/listObjects", "This operation will result in a GET call to list root objects with default paging."},
		{"Statistics", cfg.RootURL + "/stats", "This operation will result in a GET call to list root objects with default paging."},
		{"Users", cfg.RootURL + "/users", "This is a list of all users."},
	}

	data := struct {
		RootURL           string
		DistinguishedName string
		APIFunctions      []struct{ Name, RelativeLink, Description string }
	}{
		RootURL:           cfg.RootURL,
		DistinguishedName: caller.DistinguishedName,
		APIFunctions:      apiFuncs,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		h.sendErrorResponse(w, 500, err, err.Error())
		return
	}
}

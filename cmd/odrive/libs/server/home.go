package server

import (
	"errors"
	"net/http"

	cfg "decipher.com/object-drive-server/config"
	"golang.org/x/net/context"
)

// home is a method handler on AppServer for displaying a response when the
// root URI is requested without an operation. In this context, a UI is provided
// listing and linking to some available operations
func (h AppServer) home(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}

	tmpl := h.TemplateCache.Lookup("home.html")

	// Anonymous struct syntax is tricky.
	apiFuncs := []struct{ Name, RelativeLink, Description string }{
		{"List Objects", cfg.NginxRootURL + "/ui/listObjects", "This operation will result in a GET call to list root objects with default paging."},
		{"Statistics", cfg.NginxRootURL + "/stats", "This operation will result in a GET call to list root objects with default paging."},
		{"Users", cfg.NginxRootURL + "/users", "This is a list of all users."},
	}

	data := struct {
		RootURL           string
		DistinguishedName string
		APIFunctions      []struct{ Name, RelativeLink, Description string }
	}{
		RootURL:           cfg.NginxRootURL,
		DistinguishedName: caller.DistinguishedName,
		APIFunctions:      apiFuncs,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		return NewAppError(500, err, err.Error())
	}
	return nil
}

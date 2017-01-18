package server

import (
	"errors"
	"net/http"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

// home is a method handler on AppServer for displaying a response when the
// root URI is requested without an operation. In this context, a UI is provided
// listing and linking to some available operations
func (h AppServer) home(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	if h.TemplateCache == nil {
		herr := do404(ctx, w, r)
		h.publishError(gem, herr)
		return herr
	}

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		herr := NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
		h.publishError(gem, herr)
		return herr
	}

	tmpl := h.TemplateCache.Lookup("home.html")

	// Anonymous struct syntax is tricky.
	// TODO get the BaseURL off of the config
	apiFuncs := []struct{ Name, RelativeLink, Description string }{
		{"List Objects", h.Conf.BasePath + "/ui/listObjects", "This operation will result in a GET call to list root objects with default paging."},
		{"Statistics", h.Conf.BasePath + "/stats", "This operation will result in a GET call to list root objects with default paging."},
		{"API: Users", h.Conf.BasePath + "/users", "This is a list of all users via API call."},
		{"API: Objects", h.Conf.BasePath + "/objects", "This is a list of objects in users root via API call."},
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
		herr := NewAppError(500, err, err.Error())
		h.publishError(gem, herr)
		return herr
	}
	h.publishSuccess(gem, r)
	return nil
}

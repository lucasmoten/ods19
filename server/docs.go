package server

import (
	"net/http"

	"decipher.com/object-drive-server/services/audit"

	"golang.org/x/net/context"
)

// docs is a method handler on AppServer for displaying a response when the
// root URI is requested without an operation. In this context, a UI is provided
// listing and linking to some available operations
func (h AppServer) docs(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")
	if h.TemplateCache == nil {
		herr := do404(ctx, w, r)
		h.publishError(gem, herr)
		return herr
	}

	tmpl := h.TemplateCache.Lookup("home.html")

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, nil); err != nil {
		herr := NewAppError(500, err, err.Error())
		h.publishError(gem, herr)
		return herr
	}
	h.publishSuccess(gem, w)
	return nil
}

package server

import (
	"net/http"

	"golang.org/x/net/context"
)

// docs is a method handler on AppServer for displaying a response when the
// root URI is requested without an operation. In this context, a UI is provided
// listing and linking to some available operations
func (h AppServer) docs(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	if h.TemplateCache == nil {
		return do404(ctx, w, r)
	}

	tmpl := h.TemplateCache.Lookup("root.html")

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, nil); err != nil {
		return NewAppError(500, err, err.Error())
	}
	return nil
}

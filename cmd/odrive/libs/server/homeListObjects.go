package server

import (
	"errors"
	"net/http"

	cfg "decipher.com/object-drive-server/config"
	"golang.org/x/net/context"
)

func (h *AppServer) homeListObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	parentID := r.URL.Query().Get("parentId")
	tmpl := h.TemplateCache.Lookup("listObjects.html")
	data := struct{ RootURL, DistinguishedName, ParentID string }{
		RootURL:           cfg.NginxRootURL,
		DistinguishedName: caller.DistinguishedName,
		ParentID:          parentID,
	}

	tmpl.Execute(w, data)

	countOKResponse()
}

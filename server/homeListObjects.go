package server

import (
	"errors"
	"net/http"

	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h *AppServer) homeListObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
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

	parentID := r.URL.Query().Get("parentId")
	tmpl := h.TemplateCache.Lookup("listObjects.html")
	data := struct{ RootURL, DistinguishedName, ParentID string }{
		RootURL:           cfg.NginxRootURL,
		DistinguishedName: caller.DistinguishedName,
		ParentID:          parentID,
	}

	if err := tmpl.Execute(w, data); err != nil {
		herr := NewAppError(500, err, err.Error())
		h.publishError(gem, herr)
		return herr
	}
	h.publishSuccess(gem, w)
	return nil
}

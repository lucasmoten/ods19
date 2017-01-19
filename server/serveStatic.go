package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/util"
)

var (
	errStaticResourceNotFound = "Could not find static resource."
	errServingStatic          = "Error serving static file."
)

func (h AppServer) serveStatic(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	re := h.Routes.StaticFiles
	uri := r.URL.Path
	groups := util.GetRegexCaptureGroups(uri, re)
	afterStatic, ok := groups["path"]
	if !ok {
		herr := NewAppError(404, fmt.Errorf("path for static resource not given in request"), errStaticResourceNotFound)
		h.publishError(gem, herr)
		return herr
	}
	path := filepath.Join(h.StaticDir, afterStatic)
	if err := util.SanitizePath(path); err != nil {
		herr := NewAppError(404, err, errStaticResourceNotFound)
		h.publishError(gem, herr)
		return herr
	}
	w.Header().Set("Content-Type", GetContentTypeFromFilename(path))
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		herr := NewAppError(404, err, errStaticResourceNotFound)
		h.publishError(gem, herr)
		return herr
	}
	_, err = io.Copy(w, f)
	if err != nil {
		herr := NewAppError(500, err, errServingStatic)
		h.publishError(gem, herr)
		return herr
	}
	h.publishSuccess(gem, w)
	return nil
}

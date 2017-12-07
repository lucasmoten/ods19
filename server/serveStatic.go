package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/services/audit"
	"github.com/deciphernow/object-drive-server/util"
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
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		herr := NewAppError(500, errors.New("Could not get capture groups"), "No capture groups.")
		h.publishError(gem, herr)
		return herr
	}
	afterStatic := captured["path"]
	path := path.Clean(filepath.Join(h.StaticDir, afterStatic))
	// Because these are the static files that we control the names of, simply disallow escaping entirely,
	// and avoid nitpicking about combinations.  Foreign names may show up in other places, but not required
	// here.
	if strings.Contains(path, "%") || strings.Contains(path, "\\") {
		herr := NewAppError(403, fmt.Errorf("Static file paths do not allow escaping"), errServingStatic)
		h.publishError(gem, herr)
		return herr
	}
	// Sanitize path ensures that we are somewhere under the root
	if err := util.SanitizePath(h.StaticDir, path); err != nil {
		herr := NewAppError(404, err, errStaticResourceNotFound)
		h.publishError(gem, herr)
		return herr
	}
	if !strings.Contains(path, h.StaticDir) {
		herr := NewAppError(403, fmt.Errorf("path for static resource must be within the static directory"), errServingStatic)
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

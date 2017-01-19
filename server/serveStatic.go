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
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		herr := NewAppError(500, errors.New("Could not get capture groups"), "No capture groups.")
		h.publishError(gem, herr)
		return herr
	}
	afterStatic := captured["path"]
	if strings.Contains(afterStatic, "%25") || strings.Contains(afterStatic, "%") ||
		strings.Contains(afterStatic, "..") ||
		strings.Contains(afterStatic, "%2e%2e") || strings.Contains(afterStatic, "%u002e%u002e") ||
		strings.Contains(afterStatic, "%2E%2E") || strings.Contains(afterStatic, "%u002E%u002E") ||
		strings.Contains(afterStatic, "%c0%2e") || strings.Contains(afterStatic, "%e0%40%ae") || strings.Contains(afterStatic, "%c0ae") {
		herr := NewAppError(403, fmt.Errorf("path for static resource may not be encoded"), errServingStatic)
		h.publishError(gem, herr)
		return herr
	}
	path := path.Clean(filepath.Join(h.StaticDir, afterStatic))
	if err := util.SanitizePath(path); err != nil {
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

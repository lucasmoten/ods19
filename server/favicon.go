package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/deciphernow/object-drive-server/services/audit"
	"github.com/deciphernow/object-drive-server/util"
	"go.uber.org/zap"

	"golang.org/x/net/context"
)

// favicon is a method handler on AppServer for returning an icon as the
// website favicon for the path. This loads the icon file named 'favicon.ico'
// and returns it with the appropriate content type. Primarily avoids logging
// 404s for this commonly browser requested resource.
func (h AppServer) favicon(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	path := filepath.Join(h.StaticDir, "favicon.ico")
	LoggerFromContext(ctx).Info("favicon path", zap.String("path", path))
	if err := util.SanitizePath(h.StaticDir, path); err != nil {
		herr := NewAppError(404, nil, errStaticResourceNotFound)
		h.publishError(gem, herr)
		return herr
	}

	f, err := os.Open(path)
	if err != nil {
		herr := NewAppError(404, nil, errStaticResourceNotFound)
		h.publishError(gem, herr)
		return herr
	}
	defer f.Close()
	w.Header().Set("Content-Type", "image/x-icon")
	_, err = io.Copy(w, f)
	if err != nil {
		herr := NewAppError(500, nil, errServingStatic)
		h.publishError(gem, herr)
		return herr
	}
	h.publishSuccess(gem, w)

	return nil
}

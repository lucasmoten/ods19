package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"decipher.com/object-drive-server/util"
	"github.com/uber-go/zap"

	"golang.org/x/net/context"
)

// favicon is a method handler on AppServer for returning an icon as the
// website favicon for the path. This loads the icon file named 'favicon.ico'
// and returns it with the appropriate content type. Primarily avoids logging
// 404s for this commonly browser requested resource.
func (h AppServer) favicon(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	path := filepath.Join(h.StaticDir, "favicon.ico")
	LoggerFromContext(ctx).Info("favicon path", zap.String("path", path))
	if err := util.SanitizePath(path); err != nil {
		NewAppError(404, nil, errStaticResourceNotFound)
	}

	f, err := os.Open(path)
	if err != nil {
		return NewAppError(404, nil, errStaticResourceNotFound)
	}
	defer f.Close()
	w.Header().Set("Content-Type", "image/x-icon")
	_, err = io.Copy(w, f)
	if err != nil {
		return NewAppError(500, nil, errServingStatic)
	}

	return nil
}

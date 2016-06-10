package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/util"
)

var (
	errStaticResourceNotFound = "Could not find static resource."
	errServingStatic          = "Error serving static file."
)

func (h AppServer) serveStatic(
	ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	re := h.Routes.StaticFiles
	uri := r.URL.Path
	groups := util.GetRegexCaptureGroups(uri, re)
	afterStatic, ok := groups["path"]
	if !ok {
		return NewAppError(404, nil, errStaticResourceNotFound)
	}
	path := filepath.Join(h.StaticDir, afterStatic)
	if err := util.SanitizePath(path); err != nil {
		NewAppError(404, nil, errStaticResourceNotFound)
	}

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return NewAppError(404, nil, errStaticResourceNotFound)
	}
	_, err = io.Copy(w, f)
	if err != nil {
		return NewAppError(500, nil, errServingStatic)
	}

	return nil
}

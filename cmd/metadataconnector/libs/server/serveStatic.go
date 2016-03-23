package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"decipher.com/oduploader/util"
)

var (
	errStaticResourceNotFound = "Could not find static resource."
	errServingStatic          = "Error serving static file."
)

func (h AppServer) serveStatic(
	w http.ResponseWriter, r *http.Request, re *regexp.Regexp, uri string) {

	groups := util.GetRegexCaptureGroups(uri, re)
	afterStatic, ok := groups["path"]
	if !ok {
		h.sendErrorResponse(w, 404, nil, errStaticResourceNotFound)
		return
	}
	path := filepath.Join(h.StaticDir, afterStatic)
	if err := util.SanitizePath(path); err != nil {
		h.sendErrorResponse(w, 404, nil, errStaticResourceNotFound)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		h.sendErrorResponse(w, 404, nil, errStaticResourceNotFound)
		return
	}
	_, err = io.Copy(w, f)
	if err != nil {
		h.sendErrorResponse(w, 500, nil, errServingStatic)
		return
	}
}

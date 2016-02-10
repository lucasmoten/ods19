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
		http.Error(w, errStaticResourceNotFound, 404)
	}
	path := filepath.Join(h.StaticDir, afterStatic)
	if err := util.SanitizePath(path); err != nil {
		http.Error(w, errStaticResourceNotFound, 404)
	}

	f, err := os.Open(path)
	if err != nil {
		http.Error(w, errStaticResourceNotFound, 404)
		return
	}
	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, errServingStatic, 500)
	}
}

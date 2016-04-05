package server

import (
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context"
)

// favicon is a method handler on AppServer for returning an icon as the
// website favicon for the path. This loads the icon file named 'favicon.ico'
// and returns it with the appropriate content type. Primarily avoids logging
// 404s for this commonly browser requested resource.
func (h AppServer) favicon(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	icoFile, err := ioutil.ReadFile("favicon.ico")
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		sendErrorResponse(&w, 404, err, "Resource not found")
		return
	}
	w.Header().Set("Content-Type", "image/x-icon")
	_, err = w.Write(icoFile)

}

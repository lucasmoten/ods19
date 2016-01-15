package server

import (
	"io/ioutil"
	"net/http"
)

func (h AppServer) favicon(w http.ResponseWriter, r *http.Request) {

	icoFile, err := ioutil.ReadFile("favicon.ico")
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		http.Error(w, "Resource not found", 404)
	}
	w.Header().Set("Content-Type", "image/x-icon")
	_, err = w.Write(icoFile)

}

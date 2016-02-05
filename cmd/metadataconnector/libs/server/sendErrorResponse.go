package server

import (
	"log"
	"net/http"
)

func (h AppServer) sendErrorResponse(w http.ResponseWriter, code int, err error, msg string) {
	log.Printf("%s:%v", msg, err)
	http.Error(w, msg, code)
}

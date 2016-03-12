package server

import (
	"log"
	"net/http"
)

func (h AppServer) sendErrorResponse(w http.ResponseWriter, code int, err error, msg string) {
	log.Printf("httpCode %d:%s:%v", code, msg, err)
	http.Error(w, msg, code)
}

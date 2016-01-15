package server

import (
	"log"
	"net/http"
)

func (h AppServer) sendErrorResponse(w http.ResponseWriter, code int, err error, msg string) {
	log.Printf(msg+":%v", err)
	http.Error(w, msg, code)
}

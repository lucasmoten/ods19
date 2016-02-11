package server

import (
	//"encoding/json"
	//	"log"
	"net/http"
)

func (h AppServer) listUsers(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "application/json")
}

package server

import (
	"net/http"

	"encoding/json"

	"golang.org/x/net/context"
)

// userStats gets usage statistics vs a single user
func (h AppServer) userStats(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, nil, "could not get caller")
	}
	userStats, err := h.DAO.GetUserStats(caller.DistinguishedName)
	if err != nil {
		sendErrorResponse(&w, 500, err, "could not query for stats")
		return
	}
	retval, err := json.MarshalIndent(userStats, "", "  ")
	if err != nil {
		sendErrorResponse(&w, 500, err, "could not encode json")
		return
	}
	w.Write(retval)
	countOKResponse()
}

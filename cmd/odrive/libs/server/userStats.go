package server

import (
	"net/http"

	"encoding/json"

	"golang.org/x/net/context"
)

// userStats gets usage statistics vs a single user
func (h AppServer) userStats(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, nil, "could not get caller")
	}
	dao := DAOFromContext(ctx)

	userStats, err := dao.GetUserStats(caller.DistinguishedName)
	if err != nil {
		return NewAppError(500, err, "could not query for stats")
	}
	retval, err := json.MarshalIndent(userStats, "", "  ")
	if err != nil {
		return NewAppError(500, err, "could not encode json")
	}
	w.Write(retval)
	return nil
}

package server

import (
	"net/http"

	"decipher.com/object-drive-server/services/audit"

	"golang.org/x/net/context"
)

// userStats gets usage statistics vs a single user
func (h AppServer) userStats(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")
	caller, ok := CallerFromContext(ctx)
	if !ok {
		herr := NewAppError(500, nil, "could not get caller")
		h.publishError(gem, herr)
		return herr
	}
	dao := DAOFromContext(ctx)

	userStats, err := dao.GetUserStats(caller.DistinguishedName)
	if err != nil {
		herr := NewAppError(500, err, "could not query for stats")
		h.publishError(gem, herr)
		return herr
	}
	jsonResponse(w, userStats)
	h.publishSuccess(gem, r)
	return nil
}

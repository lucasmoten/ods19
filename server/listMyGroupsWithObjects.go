package server

import (
	"errors"
	"net/http"

	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"

	"golang.org/x/net/context"
)

func (h AppServer) listMyGroupsWithObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	gem, _ := GEMFromContext(ctx)
	gem.Action = "list"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	// Get groups for this user
	results, err := dao.GetGroupsForUser(user)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, errors.New("Database call failed: "), err.Error())
		h.publishError(gem, herr)
		return herr
	}

	// Response in requested format
	apiResponse := mapping.MapDAOGroupSpaceRSToProtocolGroupSpaceRS(&results)

	gem.Payload.Audit = WithResourcesFromDAOGroupSpaceRS(gem.Payload.Audit, results)

	// Output as JSON
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

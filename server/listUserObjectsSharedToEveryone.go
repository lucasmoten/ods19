package server

import (
	"net/http"

	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) listUserObjectsSharedToEveryone(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get info from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)
	snippetFields, _ := SnippetsFromContext(ctx)
	user.Snippets = snippetFields

	gem, _ := GEMFromContext(ctx)
	gem.Action = "list"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	// Parse Request
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing request")
		h.publishError(gem, herr)
		return herr
	}

	// Fetch matching objects
	results, err := dao.GetObjectsSharedToEveryone(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest))
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "GetObjectsSharedToEveryone query failed")
		h.publishError(gem, herr)
		return herr
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)

	// Caller permissions
	for objectIndex, object := range apiResponse.Objects {
		apiResponse.Objects[objectIndex] = object.WithCallerPermission(protocolCaller(caller))
	}

	gem.Payload.Audit = WithResourcesFromResultset(gem.Payload.Audit, results)

	// Output as JSON
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

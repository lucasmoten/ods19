package server

import (
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/services/audit"
)

func (h AppServer) listUserObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	gem, _ := GEMFromContext(ctx)
	gem.Action = "list"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	// Parse Request
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		herr := NewAppError(400, err, "Error parsing request")
		h.publishError(gem, herr)
		return herr
	}

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		herr := NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
		h.publishError(gem, herr)
		return herr
	}
	user.Snippets = snippetFields

	// Fetch objects for requested page
	results, err := dao.GetObjectsSharedToMe(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest))
	if err != nil {
		herr := NewAppError(500, err, "GetObjectsSharedToMe query failed")
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

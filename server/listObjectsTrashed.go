package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"

	"golang.org/x/net/context"
)

func (h AppServer) listObjectsTrashed(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	// Parse paging info
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

	// Get trash for this user
	results, err := dao.GetTrashedObjectsByUser(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest))
	if err != nil {
		herr := NewAppError(500, errors.New("Database call failed: "), err.Error())
		h.publishError(gem, herr)
		return herr
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)

	gem.Payload.Audit = WithResourcesFromResultset(gem.Payload.Audit, results)

	// Caller permissions
	for objectIndex, object := range apiResponse.Objects {
		apiResponse.Objects[objectIndex] = object.WithCallerPermission(protocolCaller(caller))
	}

	// Output as JSON
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

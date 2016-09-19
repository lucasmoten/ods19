package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) listObjectsTrashed(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	// Parse paging info
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		return NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
	}
	user.Snippets = snippetFields

	// Get trash for this user
	results, err := dao.GetTrashedObjectsByUser(user, *pagingRequest)
	if err != nil {
		return NewAppError(500, errors.New("Database call failed: "), err.Error())
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)

	// Caller permissions
	for objectIndex, object := range apiResponse.Objects {
		apiResponse.Objects[objectIndex] = object.WithCallerPermission(protocolCaller(caller))
	}

	// Output as JSON
	jsonResponse(w, apiResponse)

	return nil
}
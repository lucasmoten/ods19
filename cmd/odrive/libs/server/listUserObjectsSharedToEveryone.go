package server

import (
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) listUserObjectsSharedToEveryone(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get info from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)
	snippetFields, _ := SnippetsFromContext(ctx)
	user.Snippets = snippetFields

	// Parse Request
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Fetch matching objects
	results, err := dao.GetObjectsSharedToEveryone(user, *pagingRequest)
	if err != nil {
		return NewAppError(500, err, "GetObjectsSharedToEveryone query failed")
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

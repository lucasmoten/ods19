package server

import (
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) listUserObjectsSharedToEveryone(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get info from context
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

	// Get caller permissions
	h.buildCompositePermissionForCaller(ctx, &results)

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)
	jsonResponse(w, apiResponse)
	return nil
}

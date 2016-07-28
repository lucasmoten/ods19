package server

import (
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) listUserObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}
	dao := DAOFromContext(ctx)

	// Parse Request
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

	// Fetch objects for requested page
	results, err := dao.GetObjectsSharedToMe(user, *pagingRequest)
	if err != nil {
		return NewAppError(500, err, "GetObjectsSharedToMe query failed")
	}

	// Get caller permissions
	h.buildCompositePermissionForCaller(ctx, &results)

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)
	writeResultsetAsJSON(w, &apiResponse)
	return nil
}

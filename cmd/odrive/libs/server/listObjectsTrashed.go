package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) listObjectsTrashed(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

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

	// Get caller permissions
	h.buildCompositePermissionForCaller(ctx, &results)

	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)
	jsonResponse(w, &apiResponse)
	return nil
}

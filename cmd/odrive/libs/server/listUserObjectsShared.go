package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) listUserObjectsShared(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}

	// Parse Request
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Snippets
	snippetFields, err := h.FetchUserSnippets(ctx)
	if err != nil {
		return NewAppError(504, errors.New("Error retrieving user permissions."), err.Error())
	}
	user.Snippets = snippetFields

	// Fetch matching objects
	sharedObjectsResultSet, err := h.DAO.GetObjectsIHaveShared(user, *pagingRequest)
	if err != nil {
		return NewAppError(500, err, "GetObjectsIHaveShared query failed")
	}

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&sharedObjectsResultSet)
	writeResultsetAsJSON(w, &apiResponse)
	return nil
}
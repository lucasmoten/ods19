package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) listUserObjectsShared(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
			return
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}

	// Parse Request
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing request")
		return
	}

	// Snippets
	snippetFields, err := h.FetchUserSnippets(ctx)
	if err != nil {
		sendErrorResponse(&w, 504, errors.New("Error retrieving user permissions."), err.Error())
	}
	user.Snippets = snippetFields

	// Fetch matching objects
	sharedObjectsResultSet, err := h.DAO.GetObjectsIHaveShared(user, *pagingRequest)
	if err != nil {
		sendErrorResponse(&w, 500, err, "GetObjectsIHaveShared query failed")
		return
	}

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&sharedObjectsResultSet)
	writeResultsetAsJSON(w, &apiResponse)
	countOKResponse()
}

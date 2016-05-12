package server

import (
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) listUserObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) {

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

	// Fetch objects for requested page
	result, err := h.DAO.GetObjectsSharedToMe(user, *pagingRequest)
	if err != nil {
		sendErrorResponse(&w, 500, err, "GetObjectsSharedToMe query failed")
		return
	}

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&result)
	writeResultsetAsJSON(w, &apiResponse)
	countOKResponse()
}

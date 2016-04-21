package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) listObjectsTrashed(ctx context.Context, w http.ResponseWriter, r *http.Request) {

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

	// Parse paging info
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

	// Get trash for this user
	results, err := h.DAO.GetTrashedObjectsByUser(user, *pagingRequest)

	if err != nil {
		sendErrorResponse(&w, 500, errors.New("Database call failed: "), err.Error())
		return
	}

	// Map the response and write it out
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)
	writeResultsetAsJSON(w, &apiResponse)
	countOKResponse()
}

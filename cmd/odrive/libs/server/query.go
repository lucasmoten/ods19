package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) query(ctx context.Context, w http.ResponseWriter, r *http.Request) {

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
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing request")
		return
	}

	// Check if a filter was provided
	if len(pagingRequest.FilterSettings) == 0 {
		// Parse search phrase from the request path if there is no filter set
		if captured["searchPhrase"] == "" {
			sendErrorResponse(&w, http.StatusBadRequest, errors.New("Could not extract searchPhrase from URI"), "URI: "+r.URL.Path)
			return
		}
		searchPhrase := captured["searchPhrase"]
		// Build default filters with search phrase
		pagingRequest.FilterSettings = newNameAndDescriptionFilter(searchPhrase)
	}

	// Snippets
	snippetFields, err := h.FetchUserSnippets(ctx)
	if err != nil {
		sendErrorResponse(&w, 504, errors.New("Error retrieving user permissions."), err.Error())
	}
	user.Snippets = snippetFields

	// Perform the basic search
	results, err := h.DAO.SearchObjectsByNameOrDescription(user, *pagingRequest, false)
	if err != nil {
		sendErrorResponse(&w, 500, errors.New("Database call failed: "), err.Error())
		return
	}

	// Map the response and write it out
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)
	writeResultsetAsJSON(w, &apiResponse)
	countOKResponse()
}

func newNameAndDescriptionFilter(searchPhrase string) []protocol.FilterSetting {
	return []protocol.FilterSetting{
		{FilterField: "name", Condition: "contains", Expression: searchPhrase},
		{FilterField: "description", Condition: "contains", Expression: searchPhrase},
	}
}

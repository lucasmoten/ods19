package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) query(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	// Parse paging info
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Check if a filter was provided
	if len(pagingRequest.FilterSettings) == 0 {
		// Parse search phrase from the request path if there is no filter set
		if captured["searchPhrase"] == "" {
			return NewAppError(http.StatusBadRequest, errors.New("Could not extract searchPhrase from URI"), "URI: "+r.URL.Path)
		}
		searchPhrase := captured["searchPhrase"]
		// Build default filters with search phrase
		pagingRequest.FilterSettings = newNameAndDescriptionFilter(searchPhrase)
	}

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		return NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
	}
	user.Snippets = snippetFields

	// Perform the basic search
	results, err := dao.SearchObjectsByNameOrDescription(user, *pagingRequest, false)
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

func newNameAndDescriptionFilter(searchPhrase string) []protocol.FilterSetting {
	return []protocol.FilterSetting{
		{FilterField: "name", Condition: "contains", Expression: searchPhrase},
		{FilterField: "description", Condition: "contains", Expression: searchPhrase},
	}
}

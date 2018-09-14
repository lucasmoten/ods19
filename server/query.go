package server

import (
	"errors"
	"net/http"

	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"

	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) query(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "list"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	// Parse paging info
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing request")
		h.publishError(gem, herr)
		return herr
	}

	// Check if a filter was provided
	if len(pagingRequest.FilterSettings) == 0 {
		// Parse search phrase from the request path if there is no filter set
		if captured["searchPhrase"] == "" {
			herr := NewAppError(http.StatusBadRequest, errors.New("Could not extract searchPhrase from URI"), "URI: "+r.URL.Path)
			h.publishError(gem, herr)
			return herr
		}
		searchPhrase := captured["searchPhrase"]
		// Build default filters with search phrase
		pagingRequest.FilterSettings = newNameAndDescriptionFilter(searchPhrase)
	}

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusBadGateway, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
		h.publishError(gem, herr)
		return herr
	}
	user.Snippets = snippetFields

	// Perform the basic search
	results, err := dao.SearchObjectsByNameOrDescription(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest), true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, errors.New("Database call failed: "), err.Error())
		h.publishError(gem, herr)
		return herr
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)

	// Caller permissions
	for objectIndex, object := range apiResponse.Objects {
		apiResponse.Objects[objectIndex] = object.WithCallerPermission(protocolCaller(caller))
	}

	gem.Payload.Audit = WithResourcesFromResultset(gem.Payload.Audit, results)

	// Output as JSON
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

func newNameAndDescriptionFilter(searchPhrase string) []protocol.FilterSetting {
	return []protocol.FilterSetting{
		{FilterField: "name", Condition: "contains", Expression: searchPhrase},
		{FilterField: "description", Condition: "contains", Expression: searchPhrase},
	}
}

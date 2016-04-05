package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) query(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	//h.sendErrorResponse(w, 501, nil, "query is not yet implemented")

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
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

	// Perform the basic search
	user := models.ODUser{DistinguishedName: caller.DistinguishedName}
	results, err := h.DAO.SearchObjectsByNameOrDescription(user, *pagingRequest, false)
	if err != nil {
		sendErrorResponse(&w, 500, errors.New("Database call failed: "), err.Error())
		return
	}

	// Map the response and write it out
	response := mapping.MapODObjectResultsetToObjectResultset(&results)
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		msg := "Error marshalling response as JSON."
		log.Printf(msg+" %s", err.Error())
		sendErrorResponse(&w, 500, err, msg)
		return
	}
	w.Write(jsonData)

	countOKResponse()
}

func newNameAndDescriptionFilter(searchPhrase string) []protocol.FilterSetting {
	return []protocol.FilterSetting{
		{FilterField: "name", Condition: "contains", Expression: searchPhrase},
		{FilterField: "description", Condition: "contains", Expression: searchPhrase},
	}
}

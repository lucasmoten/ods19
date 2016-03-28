package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util"

	"golang.org/x/net/context"
)

func (h AppServer) query(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	//h.sendErrorResponse(w, 501, nil, "query is not yet implemented")

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse paging info
	pagingRequest, err := protocol.NewPagingRequestWithObjectID(r, nil, false)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
		return
	}

	// Check if a filter was provided
	if len(pagingRequest.FilterSettings) == 0 {
		// Parse search phrase from the request path if there is no filter set
		captured := util.GetRegexCaptureGroups(r.URL.Path, h.Routes.Query)
		if captured["searchPhrase"] == "" {
			h.sendErrorResponse(w, http.StatusBadRequest, errors.New("Could not extract searchPhrase from URI"), "URI: "+r.URL.Path)
			return
		}
		searchPhrase := captured["searchPhrase"]
		// Build default filters with search phrase
		pagingRequest.FilterSettings = newNameAndDescriptionFilter(searchPhrase)
	}

	// Perform the basic search
	results, err := h.DAO.SearchObjectsByNameOrDescription(caller.DistinguishedName, *pagingRequest, false)
	if err != nil {
		h.sendErrorResponse(w, 500, errors.New("Database call failed: "), err.Error())
		return
	}

	// Map the response and write it out
	response := mapping.MapODObjectResultsetToObjectResultset(&results)
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		msg := "Error marshalling response as JSON."
		log.Printf(msg+" %s", err.Error())
		h.sendErrorResponse(w, 500, err, msg)
		return
	}
	w.Write(jsonData)
}

func newNameAndDescriptionFilter(searchPhrase string) []protocol.FilterSetting {
	return []protocol.FilterSetting{
		{FilterField: "name", Condition: "contains", Expression: searchPhrase},
		{FilterField: "description", Condition: "contains", Expression: searchPhrase},
	}
}

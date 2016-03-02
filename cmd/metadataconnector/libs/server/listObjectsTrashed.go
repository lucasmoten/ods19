package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) listObjectsTrashed(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	var pagingRequest *protocol.PagingRequest
	var err error

	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	if r.Method == "GET" {
		vals := r.URL.Query()
		pr, err := protocol.NewPagingRequestFromURLValues(vals)
		if err != nil {
			h.sendErrorResponse(w, 400, err, "Bad GET request for listObjectsTrashed.")
			return
		}
		pagingRequest = &pr
	} else if r.Method == "POST" && r.Header.Get("Content-Type") == "application/json" {
		pr, err := protocol.NewPagingRequestFromJSONBody(r.Body)
		if err != nil {
			h.sendErrorResponse(w, 400, err, "Error parsing request")
			return
		}
		pagingRequest = &pr
	} else {
		h.sendErrorResponse(w, 400, errors.New("Bad Request"), "Unsupported HTTP Method.")
		return
	}

	results, err := h.DAO.GetTrashedObjectsByUser(
		"createdDate desc",
		pagingRequest.PageNumber,
		pagingRequest.PageSize,
		caller.DistinguishedName)

	if err != nil {
		h.sendErrorResponse(w, 500, errors.New("Database call failed: "), err.Error())
		return
	}

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

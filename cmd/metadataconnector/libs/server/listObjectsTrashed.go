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

func (h AppServer) listObjectsTrashed(ctx context.Context, w http.ResponseWriter, r *http.Request) {

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

	// Get trash for this user
	user := models.ODUser{DistinguishedName: caller.DistinguishedName}
	results, err := h.DAO.GetTrashedObjectsByUser(user, *pagingRequest)

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

package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) listUserObjectsShared(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request
	pagingRequest, err := protocol.NewPagingRequest(r, nil, false)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing request")
		return
	}

	// Fetch matching objects
	user := models.ODUser{DistinguishedName: caller.DistinguishedName}
	sharedObjectsResultSet, err := h.DAO.GetObjectsIHaveShared(user, *pagingRequest)
	if err != nil {
		sendErrorResponse(&w, 500, err, "GetObjectsIHaveShared query failed")
		return
	}

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&sharedObjectsResultSet)
	listUserObjectsSharedResponseAsJSON(w, r, caller, &apiResponse)
	countOKResponse()
}

func listUserObjectsSharedResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.ObjectResultset,
) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

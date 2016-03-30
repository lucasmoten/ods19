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

func (h AppServer) listUserObjectsShared(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request
	pagingRequest, err := parseListUserObjectsSharedRequest(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
		return
	}

	// Fetch matching objects
	user := models.ODUser{DistinguishedName: caller.DistinguishedName}
	sharedObjectsResultSet, err := h.DAO.GetObjectsIHaveShared(user, *pagingRequest)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "GetObjectsIHaveShared query failed")
	}

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&sharedObjectsResultSet)
	listUserObjectsSharedResponseAsJSON(w, r, caller, &apiResponse)
	return
}

func parseListUserObjectsSharedRequest(r *http.Request) (*protocol.PagingRequest, error) {
	return protocol.NewPagingRequestWithObjectID(r, nil, false)
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

package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) listUserObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) {

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

	// Fetch objects for requested page
	user := models.ODUser{DistinguishedName: caller.DistinguishedName}
	result, err := h.DAO.GetObjectsSharedToMe(user, *pagingRequest)
	if err != nil {
		sendErrorResponse(&w, 500, err, "GetObjectsSharedToMe query failed")
	}

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&result)
	listUserObjectSharesResponseAsJSON(w, r, caller, &apiResponse)
	countOKResponse()
}

func listUserObjectSharesResponseAsJSON(
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

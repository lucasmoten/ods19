package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) createFolder(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {

		sendErrorResponse(&w, http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		return
	}
	requestObject, err := parseCreateFolderRequestAsJSON(r)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error parsing JSON")
		return
	}

	// Business Logic...

	// Always set Type
	requestObject.TypeName.String = "Folder"
	requestObject.TypeName.Valid = true

	//Setup creation prerequisites, and return if we are done with the http request due to an error
	if herr := handleCreatePrerequisites(ctx, h, &requestObject); herr != nil {
		sendAppErrorResponse(&w, herr)
		return
	}

	// Add to database
	createdObject, err := h.DAO.CreateObject(&requestObject)
	if err != nil {
		sendErrorResponse(&w, 500, err, "DAO Error creating object")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToObject(&createdObject)
	createFolderResponseAsJSON(w, r, caller, &apiResponse)
	countOKResponse()
}

func parseCreateFolderRequestAsJSON(r *http.Request) (models.ODObject, error) {
	var jsonObject protocol.CreateObjectRequest
	object := models.ODObject{}
	var err error

	if r.Header.Get("Content-Type") != "application/json" {
		err = fmt.Errorf("Content-Type is '%s', expected application/json", r.Header.Get("Content-Type"))
		return object, err
	}

	// Decode to JSON
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return object, err
	}

	// Map to internal object type
	object, err = mapping.MapCreateObjectRequestToODObject(&jsonObject)
	if err != nil {
		return object, err
	}

	return object, nil
}

func createFolderResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.Object,
) {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

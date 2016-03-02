package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) createFolder(w http.ResponseWriter, r *http.Request, caller Caller) {

	// var requestObject *models.ODObject
	// var requestACM *models.ODACM
	// var err error

	if r.Header.Get("Content-Type") != "application/json" {

		h.sendErrorResponse(w, http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		return
	}
	requestObject, requestACM, err := parseCreateFolderRequestAsJSON(r)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error parsing JSON")
		return
	}

	// Business Logic...

	// Clear any passed in permission assignments on create
	//requestObject.Permissions = make([]models.ODObjectPermission, 0)
	// Always set Type
	requestObject.TypeName.String = "Folder"
	requestObject.TypeName.Valid = true

	//Setup creation prerequisites, and return if we are done with the http request due to an error
	if herr := handleCreatePrerequisites(h, &requestObject, &requestACM, w, caller); herr != nil {
		h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
        return
	}

	// Add to database
	createdObject, err := h.DAO.CreateObject(&requestObject, &requestACM)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "DAO Error creating object")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToObject(&createdObject)
	createFolderResponseAsJSON(w, r, caller, &apiResponse)

}

func parseCreateFolderRequestAsJSON(r *http.Request) (models.ODObject, models.ODACM, error) {
	var jsonObject protocol.CreateObjectRequest
	object := models.ODObject{}
	acm := models.ODACM{}
	var err error

	if r.Header.Get("Content-Type") != "application/json" {
		err = fmt.Errorf("Content-Type is '%s', expected application/json", r.Header.Get("Content-Type"))
		return object, acm, err
	}

	// Decode to JSON
	err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	if err != nil {
		return object, acm, err
	}

	// Map to internal object type
	object, err = mapping.MapCreateObjectRequestToODObject(&jsonObject)

	// TODO: Figure out how we want to pass ACM into this operation. Should it
	// be nested in protocol Object? If so, should ODObject contain ODACM ?
	return object, acm, err
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

package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
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
    if handleCreatePrerequisites(h, &requestObject, &requestACM, w, caller) {
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
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		createFolderResponseAsJSON(w, r, caller, &apiResponse)
	default:
		createFolderResponseAsHTML(w, r, caller, &apiResponse)
	}

}

func parseCreateFolderRequestAsJSON(r *http.Request) (models.ODObject, models.ODACM, error) {
	var jsonObject protocol.CreateObjectRequest
	object := models.ODObject{}
	acm := models.ODACM{}
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	case r.Header.Get("Content-Type") == "multipart/form-data":
		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			return object, acm, err
		}
		for {
			part, err := multipartReader.NextPart()
			if err != nil {
				return object, acm, err
			}
			switch {
			case part.Header.Get("Content-Type") == "application/json":

				// Read in the JSON - up to 10K
				valueAsBytes := make([]byte, 10240)
				n, err := part.Read(valueAsBytes)
				if err != nil {
					return object, acm, err
				}
				err = (json.NewDecoder(bytes.NewReader(valueAsBytes[0:n]))).Decode(&jsonObject)
			case part.Header.Get("Content-Disposition") == "form-data":
				// TODO: Maybe these header checks need to be if the value begins with?
			}
		}
	}

	// Map to internal object type
	object = mapping.MapCreateObjectRequestToODObject(&jsonObject)
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

func createFolderResponseAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.Object,
) {
	// Bounce to redraw the list
	listObjectsURL := config.RootURL
	if len(response.ParentID) > 0 {
		parentID := response.ParentID
		listObjectsURL += "/object/" + parentID + "/list"
	} else {
		listObjectsURL += "/objects"
	}
	http.Redirect(w, r, listObjectsURL, 301)
}

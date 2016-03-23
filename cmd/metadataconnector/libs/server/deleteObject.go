package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) deleteObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	var requestObject models.ODObject
	var err error

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectRequest(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing JSON")
		return
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object")
		return
	}

	// Check if the user has permissions to delete the ODObject
	//		Permission.grantee matches caller, and AllowDelete is true
	authorizedToDelete := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName &&
			permission.AllowDelete {
			authorizedToDelete = true
			break
		}
	}
	if !authorizedToDelete {
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// If the object is already deleted,
	if dbObject.IsDeleted {
		// Check its state
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 410, err, "The referenced object no longer exists.")
			return
		default:
			// NO change will be applied, but deletedDate will still be exposed in
			// the output
		}
	} else {
		// Call DAO to update the object to reflect that it is
		// deleted.  The DAO checks the changeToken and handles the child calls
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ChangeToken = requestObject.ChangeToken
		err = h.DAO.DeleteObject(dbObject, true)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "DAO Error deleting object")
			return
		}
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToDeletedObjectResponse(&dbObject)
	herr := deleteObjectResponse(w, r, &apiResponse)
	if herr != nil {
		h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
		return
	}
}

func parseDeleteObjectRequest(r *http.Request) (models.ODObject, error) {
	var jsonObject protocol.Object
	var requestObject models.ODObject
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
		if err != nil {
			return requestObject, err
		}
	}

	// Extract object ID from the URI.
	uri := r.URL.Path
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonObject.ID = uri[matchIndexes[2]:matchIndexes[3]]
			if err != nil {
				return requestObject, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	// Map to internal object type.
	requestObject, err = mapping.MapObjectToODObject(&jsonObject)
	return requestObject, err
}

func deleteObjectResponse(
	w http.ResponseWriter,
	r *http.Request,
	response *protocol.DeletedObjectResponse,
) *AppError {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		msg := "Error marshalling response as JSON"
		return &AppError{500, err, msg}
	}
	w.Write(jsonData)
	return nil
}

package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectFromTrash(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	var caller Caller
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user")
		return
	}

	// Get the change token off the request.
	changeToken, err := protocol.NewChangeTokenStructFromJSONBody(r.Body)
	if err != nil {
		sendErrorResponse(&w, http.StatusBadRequest, err, "Unexpected change token")
		return
	}

	requestObject, err := parseGetObjectRequest(ctx)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error parsing URI")
		return
	}
	originalObject, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error retrieving object from database")
		return
	}

	if originalObject.IsExpunged {
		sendErrorResponse(&w, 410, errors.New("Cannot undelete an expunged object"), "Object was expunged")
		return
	}

	if originalObject.IsAncestorDeleted {
		sendErrorResponse(&w, 405, errors.New("Cannot undelete an object with a deleted parent"), "Object has deleted ancestor")
		return
	}

	if originalObject.ChangeToken != changeToken.ChangeToken {
		sendErrorResponse(&w, http.StatusBadRequest,
			errors.New("Changetoken in database does not match client changeToken"), "Invalid changeToken.")
		return
	}

	// TODO: abstract this into a method on ODObject AuthorizedToDelete(caller)
	authorizedToDelete := false
	for _, permission := range originalObject.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowDelete {
			authorizedToDelete = true
		}
	}
	if !authorizedToDelete {
		sendErrorResponse(&w, 403, errors.New("Unauthorized for undelete"), "Unauthorized for undelete.")
		return
	}

	originalObject.ModifiedBy = caller.DistinguishedName

	// Call undelete on the DAO with the object.
	unDeletedObj, err := h.DAO.UndeleteObject(&originalObject)
	log.Println("UndeletedObject from DAO: ", unDeletedObj)

	// getproperties and return a protocol object
	resultObj := mapping.MapODObjectToObject(&unDeletedObj)
	log.Println("Undelete result: ", resultObj)

	// Write the response as JSON
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(resultObj, "", "  ")
	if err != nil {
		sendErrorResponse(&w, 500, err, "Could not marshal JSON response.")
		return
	}
	log.Println("Returning JSON response.")
	w.Write(jsonData)

	countOKResponse()
}

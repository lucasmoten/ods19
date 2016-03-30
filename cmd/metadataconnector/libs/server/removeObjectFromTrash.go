package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectFromTrash(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	var caller Caller
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user")
		return
	}

	// Get the change token off the request.
	changeToken, err := protocol.NewChangeTokenStructFromJSONBody(r.Body)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, err, "Unexpected change token")
		return
	}

	// Parse the objectID from the request URI.
	captured := util.GetRegexCaptureGroups(r.URL.Path, h.Routes.TrashObject)
	if captured["objectId"] == "" {
		h.sendErrorResponse(w, http.StatusBadRequest,
			errors.New("Could not extract objectID from URI"), "URI: "+r.URL.Path)
		return
	}

	bytesID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, err, "Invalid objectID in URI")
		return
	}

	var obj models.ODObject
	obj.ID = bytesID
	originalObject, err := h.DAO.GetObject(obj, true)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object from database")
		return
	}

	if originalObject.IsExpunged {
		h.sendErrorResponse(w, 410, errors.New("Cannot undelete an expunged object"), "Object was expunged")
		return
	}

	if originalObject.ChangeToken != changeToken.ChangeToken {
		h.sendErrorResponse(w, http.StatusBadRequest,
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
		h.sendErrorResponse(w, 403, errors.New("Unauthorized for undelete"), "Unauthorized for undelete.")
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
		h.sendErrorResponse(w, 500, err, "Could not marshal JSON response.")
		return
	}
	log.Println("Returning JSON response.")
	w.Write(jsonData)
	return
}

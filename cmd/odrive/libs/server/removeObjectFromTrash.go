package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectFromTrash(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var caller Caller
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user")
	}
	dao := DAOFromContext(ctx)

	// Get the change token off the request.
	changeToken, err := protocol.NewChangeTokenStructFromJSONBody(r.Body)
	if err != nil {
		return NewAppError(http.StatusBadRequest, err, "Unexpected change token")
	}

	requestObject, err := parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}
	originalObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object from database")
	}

	if originalObject.IsExpunged {
		return NewAppError(410, errors.New("Cannot undelete an expunged object"), "Object was expunged")
	}

	if originalObject.IsAncestorDeleted {
		return NewAppError(405, errors.New("Cannot undelete an object with a deleted parent"), "Object has deleted ancestor")
	}

	if originalObject.ChangeToken != changeToken.ChangeToken {
		return NewAppError(http.StatusBadRequest,
			errors.New("Changetoken in database does not match client changeToken"), "Invalid changeToken.")
	}

	// TODO: abstract this into a method on ODObject AuthorizedToDelete(caller)
	authorizedToDelete := false
	for _, permission := range originalObject.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowDelete {
			authorizedToDelete = true
		}
	}
	if !authorizedToDelete {
		return NewAppError(403, errors.New("Unauthorized for undelete"), "Unauthorized for undelete.")
	}

	originalObject.ModifiedBy = caller.DistinguishedName

	// Call undelete on the DAO with the object.
	unDeletedObj, err := dao.UndeleteObject(&originalObject)
	//log.Printf("UndeletedObject from DAO: %v\n", unDeletedObj)
	if err != nil {
		return NewAppError(500, err, "Error restoring object")
	}

	// getproperties and return a protocol object
	resultObj := mapping.MapODObjectToObject(&unDeletedObj)
	//log.Printf("Undelete result: %v\n", resultObj)

	// Write the response as JSON
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(resultObj, "", "  ")
	if err != nil {
		return NewAppError(500, err, "Could not marshal JSON response.")
	}
	log.Println("Returning JSON response.")
	w.Write(jsonData)

	return nil
}

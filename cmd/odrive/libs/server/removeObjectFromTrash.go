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
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object from database")
	}

	if dbObject.IsExpunged {
		return NewAppError(410, errors.New("Cannot undelete an expunged object"), "Object was expunged")
	}

	if dbObject.IsAncestorDeleted {
		return NewAppError(405, errors.New("Cannot undelete an object with a deleted parent"), "Object has deleted ancestor")
	}

	if dbObject.ChangeToken != changeToken.ChangeToken {
		return NewAppError(http.StatusBadRequest,
			errors.New("Changetoken in database does not match client changeToken"), "Invalid changeToken.")
	}

	// Check if the user has permissions to undelete the ODObject
	if ok := isUserAllowedToDelete(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to undelete this object")
	}

	// Prepare to undelete.
	dbObject.ModifiedBy = caller.DistinguishedName

	// Call undelete on the DAO with the object.
	unDeletedObj, err := dao.UndeleteObject(&dbObject)
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

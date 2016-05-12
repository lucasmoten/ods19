package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// listObjects is a method handler on AppServer for implementing the listObjects
// microservice operation.  If parentID is given in the request URI, then it is
// used to list the children within it, otherwise, the root for the given user
// is listed.  For a user, the root is defined as those objects that they own
// which have no parent identifier set.
func (h AppServer) listObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
			return
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}

	parentObject := models.ODObject{}
	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err = protocol.NewPagingRequest(r, captured, false)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing request")
		return
	}
	if len(pagingRequest.ObjectID) == 0 {
		parentObject.ID = nil
	} else {
		parentObject.ID, err = hex.DecodeString(pagingRequest.ObjectID)
		if err != nil {
			sendErrorResponse(&w, 400, err, "Object Identifier in Request URI is not a hex string")
			return
		}
	}

	// Snippets
	snippetFields, err := h.FetchUserSnippets(ctx)
	if err != nil {
		sendErrorResponse(&w, 504, errors.New("Error retrieving user permissions."), err.Error())
	}
	user.Snippets = snippetFields

	// Fetch the matching objects
	var response models.ODObjectResultset
	if parentObject.ID == nil {
		// Requesting root
		response, err = h.DAO.GetRootObjectsWithPropertiesByUser(user, *pagingRequest)
	} else {
		// Requesting children of an object. Load parent first.
		dbObject, err := h.DAO.GetObject(parentObject, false)
		if err != nil {
			log.Println(err)
			sendErrorResponse(&w, 500, err, "Error retrieving object")
			return
		}
		// Check for permission to read this object
		canReadObject := false
		for _, perm := range dbObject.Permissions {
			if perm.AllowRead && perm.Grantee == user.DistinguishedName {
				canReadObject = true
				break
			}
		}
		if !canReadObject {
			sendErrorResponse(&w, 403, err, "Insufficient permissions to list contents of this object")
			return
		}
		// Is it deleted?
		if dbObject.IsDeleted {
			switch {
			case dbObject.IsExpunged:
				sendErrorResponse(&w, 410, err, "The object no longer exists.")
				return
			case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
				sendErrorResponse(&w, 405, err, "The object cannot be read because an ancestor is deleted.")
				return
			case dbObject.IsDeleted:
				sendErrorResponse(&w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its contents")
				return
			}
		}

		// Get the objects
		response, err = h.DAO.GetChildObjectsWithPropertiesByUser(user, *pagingRequest, parentObject)
	}
	if err != nil {
		log.Println(err)
		sendErrorResponse(&w, 500, err, "General error")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
	writeResultsetAsJSON(w, &apiResponse)
	countOKResponse()
}

func writeResultsetAsJSON(
	w http.ResponseWriter,
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

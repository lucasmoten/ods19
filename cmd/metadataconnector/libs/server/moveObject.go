package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) moveObject(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	var requestObject models.ODObject
	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request in sent format
	if r.Header.Get("Content-Type") != "application/json" {
		sendErrorResponse(&w, http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		return
	}
	requestObject, err = parseMoveObjectRequestAsJSON(r, ctx)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing JSON")
		return
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error retrieving object")
		return
	}

	// Capture and overwrite here for comparison later after the update
	requestObject.ChangeCount = dbObject.ChangeCount

	// Check if the user has permissions to update the ODObject
	//		Permission.grantee matches caller, and AllowUpdate is true
	authorizedToUpdate := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName &&
			permission.AllowRead && permission.AllowUpdate {
			authorizedToUpdate = true
			break
		}
	}
	if !authorizedToUpdate {
		sendErrorResponse(&w, 403, nil, "Unauthorized")
		return
	}

	// Check if the user has permission to create children under the target
	// object for which they are moving this one to (the parentID)
	targetParent := models.ODObject{}
	targetParent.ID = requestObject.ParentID
	dbParent, err := h.DAO.GetObject(targetParent, false)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error retrieving parent to move object into")
		return
	}
	authorizedToMoveTo := false
	for _, parentPermission := range dbParent.Permissions {
		if parentPermission.Grantee == caller.DistinguishedName &&
			parentPermission.AllowRead && parentPermission.AllowCreate {
			authorizedToMoveTo = true
			break
		}
	}
	if !authorizedToMoveTo {
		sendErrorResponse(&w, 403, nil, "Unauthorized")
		// log this, but done send back to client as it leaks existence
		log.Printf("User has insufficient permissions to move object into new parent")
		return
	}

	// Parent must not be deleted
	if targetParent.IsDeleted {
		if targetParent.IsExpunged {
			sendErrorResponse(&w, 410, err, "Unable to move object into an object that does not exist")
			return
		}
		sendErrorResponse(&w, 405, err, "Unable to move object into an object that is deleted")
		return
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			sendErrorResponse(&w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			sendErrorResponse(&w, 405, err, "The object cannot be modified because an ancestor is deleted.")
			return
		case dbObject.IsDeleted:
			sendErrorResponse(&w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
			return
		}
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		sendErrorResponse(&w, 428, nil, "ChangeToken does not match expected value. Object may have been changed by another request.")
		return
	}

	// #60 Check that the parent being assigned for the object passed in does not
	// result in a circular reference
	if bytes.Compare(requestObject.ParentID, requestObject.ID) == 0 {
		sendErrorResponse(&w, 400, err, "ParentID cannot be set to the ID of the object. Circular references are not allowed.")
		return
	}
	circular, err := h.DAO.IsParentIDADescendent(requestObject.ID, requestObject.ParentID)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error retrieving ancestor to check for circular references")
		return
	}
	if circular {
		sendErrorResponse(&w, 400, err, "ParentID cannot be set to the value specified as would result in a circular reference")
		return
	}

	// Check that the parent of the object passed in is different then the current
	// state of the object in the data store
	if bytes.Compare(requestObject.ParentID, dbObject.ParentID) == 0 {
		// NOOP, will return current state
		requestObject = dbObject
	} else {

		// Call metadata connector to update the object in the data store
		// We reference the dbObject here instead of request to isolate what is
		// allowed to be changed in this operation
		// Force the modified by to be that of the caller
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ParentID = requestObject.ParentID
		err := h.DAO.UpdateObject(&dbObject)
		if err != nil {
			log.Printf("Error updating object: %v", err)
			sendErrorResponse(&w, 500, nil, "Error saving object in new location")
			return
		}

		// After the update, check that key values have changed...
		if dbObject.ChangeCount <= requestObject.ChangeCount {
			sendErrorResponse(&w, 500, nil, "ChangeCount didn't update when processing move request")
			return
		}
		if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
			sendErrorResponse(&w, 500, nil, "ChangeToken didn't update when processing move request")
			return
		}
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToObject(&dbObject)
	moveObjectResponseAsJSON(w, r, caller, &apiResponse)

	countOKResponse()
}

func parseMoveObjectRequestAsJSON(r *http.Request, ctx context.Context) (models.ODObject, error) {
	var jsonObject protocol.Object
	var requestObject models.ODObject
	var err error

	// Depends on this for the changeToken
	err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	if err != nil {
		return requestObject, err
	}

	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return requestObject, errors.New("Could not get capture groups")
	}

	// Initialize requestobject with the objectId being requested
	if captured["objectId"] == "" {
		return requestObject, errors.New("Could not extract ObjectID from URI")
	}
	_, err = hex.DecodeString(captured["objectId"])
	if err != nil {
		return requestObject, errors.New("Invalid ObjectID in URI")
	}
	jsonObject.ID = captured["objectId"]
	// And the new folderId
	if len(captured["folderId"]) > 0 {
		_, err = hex.DecodeString(captured["folderId"])
		if err != nil {
			return requestObject, errors.New("Invalid flderId in URI")
		}
		jsonObject.ParentID = captured["folderId"]
	}

	// Map to internal object type
	requestObject, err = mapping.MapObjectToODObject(&jsonObject)
	return requestObject, err
}

func moveObjectResponseAsJSON(
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

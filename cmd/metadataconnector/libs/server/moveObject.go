package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) moveObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	var requestObject models.ODObject
	var err error

	// Parse Request in sent format
	if r.Header.Get("Content-Type") != "application/json" {

		h.sendErrorResponse(w, http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		return
	}
	requestObject, err = parseMoveObjectRequestAsJSON(r)
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
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// Check if the user has permission to create children under the target
	// object for which they are moving this one to (the parentID)
	targetParent := models.ODObject{}
	targetParent.ID = requestObject.ParentID
	dbParent, err := h.DAO.GetObject(targetParent, false)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error retrieving parent to move object into")
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
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		// log this, but done send back to client as it leaks existence
		log.Printf("User has insufficient permissions to move object into new parent")
	}

	// Parent must not be deleted
	if targetParent.IsDeleted {
		if targetParent.IsExpunged {
			h.sendErrorResponse(w, 410, err, "Unable to move object into an object that does not exist")
			return
		}
		h.sendErrorResponse(w, 405, err, "Unable to move object into an object that is deleted")
		return
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "The object cannot be modified because an ancestor is deleted.")
			return
		case dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
			return
		}
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		h.sendErrorResponse(w, 428, nil, "ChangeToken does not match expected value. Object may have been changed by another request.")
		return
	}

	// #60 Check that the parent being assigned for the object passed in does not
	// result in a circular reference
	if bytes.Compare(requestObject.ParentID, requestObject.ID) == 0 {
		h.sendErrorResponse(w, 400, err, "ParentID cannot be set to the ID of the object. Circular references are not allowed.")
		return
	}
	circular, err := h.DAO.IsParentIDADescendent(requestObject.ID, requestObject.ParentID)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error retrieving ancestor to check for circular references")
		return
	}
	if circular {
		h.sendErrorResponse(w, 400, err, "ParentID cannot be set to the value specified as would result in a circular reference")
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
			//TODO: should we not send an error response in this case?
			log.Printf("Error updating object: %v", err)
		}

		// After the update, check that key values have changed...
		if dbObject.ChangeCount <= requestObject.ChangeCount {
			h.sendErrorResponse(w, 500, nil, "ChangeCount didn't update when processing move request")
			return
		}
		if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
			h.sendErrorResponse(w, 500, nil, "ChangeToken didn't update when processing move request")
			return
		}
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToObject(&dbObject)
	moveObjectResponseAsJSON(w, r, caller, &apiResponse)

}

func parseMoveObjectRequestAsJSON(r *http.Request) (models.ODObject, error) {
	var jsonObject protocol.Object
	var requestObject models.ODObject
	var err error

	// Depends on this for the changeToken
	err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	if err != nil {
		return requestObject, err
	}

	// Portions from the request URI itself ...
	uri := r.URL.Path
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)/move/([0-9a-fA-F]*)")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonObject.ID = uri[matchIndexes[2]:matchIndexes[3]]
			if err != nil {
				return requestObject, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
		if len(matchIndexes) > 5 {
			jsonObject.ParentID = uri[matchIndexes[4]:matchIndexes[5]]
			if err != nil {
				return requestObject, errors.New("Parent Identifier in Request URI is not a hex string")
			}
		}
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

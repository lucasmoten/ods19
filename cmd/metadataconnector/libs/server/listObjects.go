package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

// listObjects is a method handler on AppServer for implementing the listObjects
// microservice operation.  If parentID is given in the request URI, then it is
// used to list the children within it, otherwise, the root for the given user
// is listed.  For a user, the root is defined as those objects that they own
// which have no parent identifier set.
// Request format:
//				GET /services/object-drive/object/{objectId}/list HTTP/1.1
//				Host: fully.qualified.domain.name
//				Content-Type: application/json;
//				Content-Length: nnn
//
//				{
//					"pageNumber": "{pageNumber}",
//					"pageSize": {pageSize}
//				}
func (h AppServer) listObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	parentObject := models.ODObject{}
	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request
	pagingRequest, err = parseListObjectsRequest(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
		return
	}
	if len(pagingRequest.ObjectID) == 0 {
		parentObject.ID = nil
	} else {
		parentObject.ID, err = hex.DecodeString(pagingRequest.ObjectID)
		if err != nil {
			h.sendErrorResponse(w, 400, err, "Object Identifier in Request URI is not a hex string")
			return
		}
	}

	// Fetch the matching objects
	var response models.ODObjectResultset
	if parentObject.ID == nil {
		// Requesting root
		response, err = h.DAO.GetRootObjectsWithPropertiesByUser(
			"createddate desc",
			pagingRequest.PageNumber,
			pagingRequest.PageSize,
			caller.DistinguishedName,
		)
	} else {
		// Requesting children of an object. Load parent first.
		dbObject, err := h.DAO.GetObject(parentObject, false)
		if err != nil {
			log.Println(err)
			h.sendErrorResponse(w, 500, err, "Error retrieving object")
			return
		}
		// Check for permission to read this object
		canReadObject := false
		for _, perm := range dbObject.Permissions {
			if perm.AllowRead && perm.Grantee == caller.DistinguishedName {
				canReadObject = true
				break
			}
		}
		if !canReadObject {
			h.sendErrorResponse(w, 403, err, "Insufficient permissions to list contents of this object")
			return
		}
		// Is it deleted?
		if dbObject.IsDeleted {
			switch {
			case dbObject.IsExpunged:
				h.sendErrorResponse(w, 410, err, "The object no longer exists.")
				return
			case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "The object cannot be read because an ancestor is deleted.")
				return
			case dbObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its contents")
				return
			}
		}

		// Get the objects
		response, err = h.DAO.GetChildObjectsWithPropertiesByUser(
			"createddate desc",
			pagingRequest.PageNumber,
			pagingRequest.PageSize,
			parentObject,
			caller.DistinguishedName,
		)
	}
	if err != nil {
		log.Println(err)
		h.sendErrorResponse(w, 500, err, "General error")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
	listObjectsResponseAsJSON(w, r, &apiResponse)
	return
}

func parseListObjectsRequest(r *http.Request) (*protocol.PagingRequest, error) {
	re := regexp.MustCompile("/object/([0-9a-fA-F]*)/list")
	return protocol.NewPagingRequestWithObjectID(r, re, false)
}

func listObjectsResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
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

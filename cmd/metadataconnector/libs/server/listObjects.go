package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"

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
func (h AppServer) listObjects(w http.ResponseWriter, r *http.Request, caller Caller) {

	parentObject := models.ODObject{}
	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request
	pagingRequest, err = parseListObjectsRequest(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
		return
	}
	if len(pagingRequest.ParentID) == 0 {
		parentObject.ID = nil
	} else {
		parentObject.ID, _ = hex.DecodeString(pagingRequest.ParentID)
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
	listObjectsResponseAsJSON(w, r, caller, &apiResponse)
	return
}

func parseListObjectsRequest(r *http.Request) (*protocol.PagingRequest, error) {
	var jsonPaging protocol.PagingRequest
	defaultPage := 1
	defaultPageSize := 20
	jsonPaging.PageNumber = defaultPage
	jsonPaging.PageSize = defaultPageSize
	var err error

	err = (json.NewDecoder(r.Body)).Decode(&jsonPaging)
	if err != nil {
		// If there is no body, it's an EOF. So report other errors
		if err != io.EOF {
			log.Printf("Error parsing paging information in json: %v", err)
			return &jsonPaging, err
		}
		// EOF ok. Reassign defaults and reset the error
		jsonPaging.PageNumber = defaultPage
		jsonPaging.PageSize = defaultPageSize
		err = nil
	}

	// Portions from the request path itself to pick up object ID to list children
	// Note that a call to /objects will not match, and hence the ID wont be set
	uri := r.URL.Path
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)/list")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonPaging.ParentID = uri[matchIndexes[2]:matchIndexes[3]]
			_, err := hex.DecodeString(jsonPaging.ParentID)
			if err != nil {
				return &jsonPaging, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	// Paging provided as querystring arguments
	sPageNumber := r.URL.Query().Get("PageNumber")
	sPageSize := r.URL.Query().Get("PageSize")
	pageNumber, errPageNumber := strconv.Atoi(sPageNumber)
	if errPageNumber == nil && pageNumber > 0 {
		jsonPaging.PageNumber = pageNumber
	}
	pageSize, errPageSize := strconv.Atoi(sPageSize)
	if errPageSize == nil && pageSize > 0 {
		jsonPaging.PageSize = pageSize
	}
	if jsonPaging.PageNumber <= 0 {
		jsonPaging.PageNumber = defaultPage
	}
	if jsonPaging.PageSize <= 0 {
		jsonPaging.PageSize = defaultPageSize
	}

	return &jsonPaging, err
}

func listObjectsResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.ObjectResultset,
) {
	// TODO: Caller passed but not used.
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

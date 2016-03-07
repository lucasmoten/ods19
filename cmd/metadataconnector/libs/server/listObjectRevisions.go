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

// listObjectRevisions is a method handler on AppServer for implementing the
// listObjectRevisions microservice operation.  The unique identifier of the
// object for which revisions are saught is presented in the path in the
// following format:
//      GET /services/object-drive/object/{objectId}/history HTTP/1.1
// Paging information can be specified in either JSON body or as querystring
// arguments:
//      GET /services/object-drive/object/{objectId}/history?PageNumber=1&PageSize=20 HTTP/1.1
// If provided in the body as an application/json, the format would look
// like this:
//		POST /services/object-drive/object/{objectId}/history HTTP/1.1
//		Host: fully.qualified.domain.name
//		Content-Type: application/json;
//		Content-Length: nnn
//
//		{
//			"pageNumber": 1,
//			"pageSize": 20
//		}
func (h AppServer) listObjectRevisions(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request
	pagingRequest, err := parseListObjectRevisions(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
		return
	}

	// Fetch matching object
	obj := models.ODObject{}
	// valid decoding checked when parsed, no need to check for error again
	obj.ID, _ = hex.DecodeString(pagingRequest.ObjectID)
	dbObject, err := h.DAO.GetObject(obj, false)
	if err != nil {
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

	// Get the revision information for this objects
	response, err := h.DAO.GetObjectRevisionsWithPropertiesByUser(
		"createddate desc",
		pagingRequest.PageNumber,
		pagingRequest.PageSize,
		dbObject,
		caller.DistinguishedName,
	)
	if err != nil {
		log.Println(err)
		h.sendErrorResponse(w, 500, err, "General error")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
	listObjectRevisionsResponseAsJSON(w, r, &apiResponse)
	return
}

func parseListObjectRevisions(r *http.Request) (*protocol.PagingRequest, error) {
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)/history")
	return protocol.NewPagingRequestWithObjectID(r, re, true)
}

func listObjectRevisionsResponseAsJSON(
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

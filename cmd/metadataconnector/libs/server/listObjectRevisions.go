package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"

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
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, captured, true)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing request")
		return
	}

	// Fetch matching object
	obj := models.ODObject{}
	// valid decoding checked when parsed, no need to check for error again
	obj.ID, err = hex.DecodeString(pagingRequest.ObjectID)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Object Identifier in Request URI is not a hex string")
		return
	}
	dbObject, err := h.DAO.GetObject(obj, false)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error retrieving object")
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

	// Get the revision information for this objects
	user := models.ODUser{DistinguishedName: caller.DistinguishedName}
	response, err := h.DAO.GetObjectRevisionsWithPropertiesByUser(user, *pagingRequest, dbObject)
	if err != nil {
		log.Println(err)
		sendErrorResponse(&w, 500, err, "General error")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
	listObjectRevisionsResponseAsJSON(w, r, &apiResponse)
	countOKResponse()
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

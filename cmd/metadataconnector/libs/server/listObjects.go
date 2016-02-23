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

// listObjects is a method handler on AppServer for implementing the listObjects
// microservice operation.  If an ID is given in the request URI, then it is
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
// TODO: Implement proper paging and and result information
// TODO: Convert response to JSON
func (h AppServer) listObjects(w http.ResponseWriter, r *http.Request, caller Caller) {

	var parentObject *models.ODObject
	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		parentObject, pagingRequest, err = parseListObjectsRequestAsJSON(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing JSON")
			return
		}
	default:
		h.sendErrorResponse(w, 500, err, "Unsupported request type. Send application/json.")
	}

	// TODO better way to handle JS passing empty string?
	if string(parentObject.ID) == "" {
		parentObject.ID = nil
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

//XXX Note that you don't need multipart/form-data for anything that won't be uploading files.
//Just leave off enctype for a trivial parameter encoding to make this ugly multipart parse go away.
func parseListObjectsRequestAsJSON(r *http.Request) (*models.ODObject, *protocol.PagingRequest, error) {
	var jsonObject protocol.Object
	var jsonPaging protocol.PagingRequest
	jsonPaging.PageNumber = 1
	jsonPaging.PageSize = 20
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonPaging)
		if err != nil {
			// TODO: log it?
			// but this is paging, so goto defaults and reset the Error
			jsonPaging.PageNumber = 1
			jsonPaging.PageSize = 20
			err = nil
		}
	case r.Header.Get("Content-Type") == "multipart/form-data":
		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			// TODO: log it?
			// but this is paging, so goto defaults and reset the Error
			jsonPaging.PageNumber = 1
			jsonPaging.PageSize = 20
			err = nil
		} else {
			for {
				part, err := multipartReader.NextPart()
				if err != nil {
					// TODO: log it?
					// but this is paging, so goto defaults and reset the Error
					jsonPaging.PageNumber = 1
					jsonPaging.PageSize = 20
					err = nil
				} else {
					switch {
					case part.Header.Get("Content-Type") == "application/json":

						// Read in the JSON - up to 10K
						valueAsBytes := make([]byte, 10240)
						n, err := part.Read(valueAsBytes)
						if err != nil {
							// TODO: log it?
							// but this is paging, so goto defaults and reset the Error
							jsonPaging.PageNumber = 1
							jsonPaging.PageSize = 20
							err = nil
						} else {
							err = (json.NewDecoder(bytes.NewReader(valueAsBytes[0:n]))).Decode(&jsonPaging)
							if err != nil {
								// TODO: log it?
								// but this is paging, so goto defaults and reset the Error
								jsonPaging.PageNumber = 1
								jsonPaging.PageSize = 20
								err = nil
							}
						}
					case part.Header.Get("Content-Disposition") == "form-data":
						// TODO: Maybe these header checks need to be if the value begins with?
						// Will we ever use this? We are not posting a new object.
					}
				}
			}
		}
	}

	// Portions from the request URI itself ...
	uri := r.URL.RequestURI()
	re, _ := regexp.Compile("/object/(.*)/list")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonObject.ID = uri[matchIndexes[2]:matchIndexes[3]]
			if err != nil {
				return nil, nil, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	// Map to internal object type
	object := mapping.MapObjectToODObject(&jsonObject)
	return &object, &jsonPaging, err
}

func listObjectsResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
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

func extractCNfromDN(dn string) (cn string) {
	cn = dn[strings.Index(dn, "=")+1 : strings.Index(dn, ",")]
	return
}

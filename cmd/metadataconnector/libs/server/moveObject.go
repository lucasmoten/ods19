package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func (h AppServer) moveObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	var requestObject *models.ODObject
	var err error

	// Parse Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		requestObject, err = parseMoveObjectRequestAsJSON(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing JSON")
			return
		}
	default:
		h.sendErrorResponse(w, 501, nil, "Reading from HTML form post not supported")
		requestObject = parseMoveObjectRequestAsHTML(r)
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(h.MetadataDB, requestObject, true)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object")
		return
	}

	// TODO
	// Check AAC to compare user clearance to NEW metadata Classifications
	// 		Check if Classification is allowed for this User

	// Check if the user has permissions to update the ODObject
	//		Permission.grantee matches caller, and AllowUpdate is true
	authorizedToUpdate := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowUpdate {
			authorizedToUpdate = true
		}
	}
	if !authorizedToUpdate {
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// Check if the user has permission to create children under the target
	// object for which they are moving this one to (the parentID)
	var targetParent *models.ODObject
	targetParent.ID = requestObject.ParentID
	targetParent, err = dao.GetObject(h.MetadataDB, targetParent, false)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error retrieving parent to move object into")
		return
	}
	authorizedToMoveTo := false
	for _, parentPermission := range targetParent.Permissions {
		if parentPermission.Grantee == caller.DistinguishedName && parentPermission.AllowCreate {
			authorizedToMoveTo = true
		}
	}
	if !authorizedToMoveTo {
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		// log this, but done send back to client as it leaks existence
		log.Printf("User has insufficient permissions to move object into new parent")
	}
	// parent must not be deleted
	if targetParent.IsDeleted {
		if targetParent.IsExpunged {
			h.sendErrorResponse(w, 400, err, "Unable to move object into an object that does not exist")
			return
		}
		h.sendErrorResponse(w, 400, err, "Unable to move object into an object that is deleted")
		return
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 400, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			h.sendErrorResponse(w, 400, err, "The object cannot be modified because an ancestor is deleted.")
			return
		case dbObject.IsDeleted:
			h.sendErrorResponse(w, 400, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
			return
		}
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		h.sendErrorResponse(w, 428, nil, "ChangeToken does not match expected value. Object may have been changed by another request.")
		return
	}

	// Check that the parent of the object passed in is different then the current
	// state of the object in the data store
	if bytes.Compare(requestObject.ParentID, dbObject.ParentID) == 0 {
		// NOOP, will return current state
		requestObject = dbObject
	} else {

		// TODO: Handle ACM. This needs to be parsed as separate object? Maybe the
		// model should have it nested like has been changed in the wiki page

		// Call metadata connector to update the object in the data store
		// We reference the dbObject here instead of request to isolate what is
		// allowed to be changed in this operation
		// Force the modified by to be that of the caller
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ParentID = requestObject.ParentID
		dao.UpdateObject(h.MetadataDB, dbObject, nil)

		// After the update, check that key values have changed...
		if requestObject.ChangeCount <= dbObject.ChangeCount {
			h.sendErrorResponse(w, 500, nil, "ChangeCount didn't update when processing request")
			return
		}
		if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
			h.sendErrorResponse(w, 500, nil, "ChangeToken didn't update when processing request")
			return
		}
	}

	// Response in requested format
	switch {
	case r.Header.Get("Content-Type") == "multipart/form-data":
		fallthrough
	case r.Header.Get("Content-Type") == "application/json":
		moveObjectResponseAsJSON(w, r, caller, dbObject)
	default:
		moveObjectResponseAsHTML(w, r, caller, dbObject)
	}

}

func parseMoveObjectRequestAsJSON(r *http.Request) (*models.ODObject, error) {
	var jsonObject models.ODObject
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	case r.Header.Get("Content-Type") == "multipart/form-data":
		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			return nil, err
		}
		for {
			part, err := multipartReader.NextPart()
			if err != nil {
				return nil, err
			}
			switch {
			case part.Header.Get("Content-Type") == "application/json":

				// Read in the JSON - up to 10K
				valueAsBytes := make([]byte, 10240)
				n, err := part.Read(valueAsBytes)
				if err != nil {
					return nil, err
				}
				err = (json.NewDecoder(bytes.NewReader(valueAsBytes[0:n]))).Decode(&jsonObject)
			case part.Header.Get("Content-Disposition") == "form-data":
				// TODO: Maybe these header checks need to be if the value begins with?
			}
		}
	}

	return &jsonObject, err
}
func parseMoveObjectRequestAsHTML(r *http.Request) *models.ODObject {
	return nil
}

func moveObjectResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObject,
) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

func moveObjectResponseAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObject,
) {

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "updateObject", caller.DistinguishedName)

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)

	fmt.Fprintf(w, pageTemplateEnd)

}

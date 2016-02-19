package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) createFolder(w http.ResponseWriter, r *http.Request, caller Caller) {

	var requestObject *models.ODObject
	var requestACM *models.ODACM
	var err error

	// Parse Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		requestObject, requestACM, err = parseCreateFolderRequestAsJSON(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing JSON")
			return
		}
	default:
		requestObject, requestACM, err = parseCreateFolderRequestAsHTML(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing HTML Form")
			return
		}
	}
	// Old UI
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, pageTemplateStart, "createFolder", caller.DistinguishedName)
		fmt.Fprintf(w, pageTemplateEnd)
		return
	}

	// Business Logic...

	// Clear any passed in permission assignments on create
	//requestObject.Permissions = make([]models.ODObjectPermission, 0)
	// Always set Type
	requestObject.TypeName.String = "Folder"
	requestObject.TypeName.Valid = true

	// Check if parent defined
	if requestObject.ParentID == nil {
		// No parent set, but need to setup permission
		newPermission := models.ODObjectPermission{}
		newPermission.Grantee = caller.DistinguishedName
		newPermission.AllowCreate = true
		newPermission.AllowRead = true
		newPermission.AllowUpdate = true
		newPermission.AllowDelete = true
		requestObject.Permissions = append(requestObject.Permissions, newPermission)
	} else {
		// Parent is defined, retrieve existing parent object from the data store

		parentObject := models.ODObject{}
		parentObject.ID = requestObject.ParentID
		dbParentObject, err := h.DAO.GetObject(&parentObject, false)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error retrieving parent object")
			return
		}

		// Check if the user has permissions to create child objects under the
		// parent.
		//		Permission.grantee matches caller, and AllowCreate is true
		authorizedToCreate := false
		for _, permission := range dbParentObject.Permissions {
			if permission.Grantee == caller.DistinguishedName &&
				permission.AllowRead && permission.AllowCreate {
				authorizedToCreate = true
				break
			}
		}
		if !authorizedToCreate {
			h.sendErrorResponse(w, 403, nil, "Unauthorized")
			return
		}

		// Make sure the object isn't deleted. To remove an object from the trash,
		// use removeObjectFromTrash call.
		if dbParentObject.IsDeleted {
			switch {
			case dbParentObject.IsExpunged:
				h.sendErrorResponse(w, 410, err, "The object no longer exists.")
				return
			case dbParentObject.IsAncestorDeleted && !dbParentObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "Unallowed to create child objects under a deleted object.")
				return
			case dbParentObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "The object under which this object is being created is currently in the trash. Use removeObjectFromTrash to restore it first.")
				return
			}
		}

		// Copy permissions from parent into request Object
		for _, permission := range dbParentObject.Permissions {
			if !permission.IsDeleted {
				newPermission := models.ODObjectPermission{}
				newPermission.Grantee = permission.Grantee
				isCreator := (permission.Grantee == caller.DistinguishedName)
				newPermission.AllowCreate = permission.AllowCreate || isCreator
				newPermission.AllowRead = permission.AllowRead || isCreator
				newPermission.AllowUpdate = permission.AllowUpdate || isCreator
				newPermission.AllowDelete = permission.AllowDelete || isCreator
				requestObject.Permissions = append(requestObject.Permissions, newPermission)
			}
		}
	}

	// Disallow creating as deleted
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		h.sendErrorResponse(w, 428, nil, "Creating object in a deleted state is not allowed")
		return
	}

	// Setup meta data...
	requestObject.CreatedBy = caller.DistinguishedName
	requestObject.OwnedBy.String = caller.DistinguishedName
	requestObject.OwnedBy.Valid = true
	requestACM.CreatedBy = caller.DistinguishedName

	// Add to database
	err = h.DAO.CreateObject(requestObject, requestACM)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "DAO Error updating object")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToObject(requestObject)
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		createFolderResponseAsJSON(w, r, caller, &apiResponse)
	default:
		createFolderResponseAsHTML(w, r, caller, &apiResponse)
	}

}

func parseCreateFolderRequestAsJSON(r *http.Request) (*models.ODObject, *models.ODACM, error) {
	var jsonObject protocol.Object
	acm := models.ODACM{}
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	case r.Header.Get("Content-Type") == "multipart/form-data":
		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			return nil, &acm, err
		}
		for {
			part, err := multipartReader.NextPart()
			if err != nil {
				return nil, &acm, err
			}
			switch {
			case part.Header.Get("Content-Type") == "application/json":

				// Read in the JSON - up to 10K
				valueAsBytes := make([]byte, 10240)
				n, err := part.Read(valueAsBytes)
				if err != nil {
					return nil, &acm, err
				}
				err = (json.NewDecoder(bytes.NewReader(valueAsBytes[0:n]))).Decode(&jsonObject)
			case part.Header.Get("Content-Disposition") == "form-data":
				// TODO: Maybe these header checks need to be if the value begins with?
			}
		}
	}

	// Map to internal object type
	object := mapping.MapObjectToODObject(&jsonObject)
	// TODO: Figure out how we want to pass ACM into this operation. Should it
	// be nested in protocol Object? If so, should ODObject contain ODACM ?
	return &object, &acm, err
}
func parseCreateFolderRequestAsHTML(r *http.Request) (*models.ODObject, *models.ODACM, error) {
	object := models.ODObject{}
	acm := models.ODACM{}
	var err error

	if r.Method == "POST" {

		// Assume root
		parentID := ""

		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			return &object, &acm, err
		} // if err != nil
		for {
			part, err := multipartReader.NextPart()
			if err != nil {
				if err == io.EOF {
					break //just an eof...not an error
				} else {
					return &object, &acm, err
				}
			} // if err != nil

			switch {
			case part.FormName() == "title":
				fallthrough
			case part.FormName() == "name":
				object.Name = getFormValueAsString(part)
			case part.FormName() == "type":
				object.TypeName.String = getFormValueAsString(part)
				object.TypeName.Valid = (len(object.TypeName.String) > 0)
			case part.FormName() == "parentId":
				parentID = getFormValueAsString(part)
				if parentID != "" {
					object.ParentID, err = hex.DecodeString(parentID)
					if err != nil {
						// failed to decode from string to []binary
						return &object, &acm, err
					}
				}
			case part.FormName() == "classification":
				acm.Classification.String = getFormValueAsString(part)
				acm.Classification.Valid = (len(acm.Classification.String) > 0)
			} // switch
		} //for
	}
	return &object, &acm, err
}

func createFolderResponseAsJSON(
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

func createFolderResponseAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.Object,
) {
	// Bounce to redraw the list
	listObjectsURL := config.RootURL
	if len(response.ParentID) > 0 {
		parentID := hex.EncodeToString(response.ParentID)
		listObjectsURL += "/object/" + parentID + "/list"
	} else {
		listObjectsURL += "/objects"
	}
	http.Redirect(w, r, listObjectsURL, 301)
}

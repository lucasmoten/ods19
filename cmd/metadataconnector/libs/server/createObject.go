package server

import (
	//"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/metadata/models/acm"
	"decipher.com/oduploader/protocol"
)

// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(ctx context.Context, w http.ResponseWriter,
	r *http.Request) {
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	var obj models.ODObject
	var createdObject models.ODObject
	var grant models.ODObjectPermission
	var err error
	var drainFunc func()

	grant.Grantee = caller.DistinguishedName
	grant.AllowRead = true
	grant.AllowCreate = true
	grant.AllowUpdate = true
	grant.AllowDelete = true
	grant.AllowShare = true
	grant.EncryptKey = utils.CreateKey()

	// Determine if this request is being made with a file stream or without.
	// When a filestream is provided, there is a different handler that parses
	// the multipart form data
	isMultipart := isMultipartFormData(r)
	if isMultipart {

		rName := utils.CreateRandomName()
		iv := utils.CreateIV()
		obj.ContentConnector.String = rName
		obj.EncryptIV = iv

		multipartReader, err := r.MultipartReader()
		if err != nil {
			sendErrorResponse(&w, 400, err, "Unable to get mime multipart")
			return
		}
		createdFunc, herr, err := h.acceptObjectUpload(ctx, multipartReader, &obj, &grant, true)
		if herr != nil {
			sendAppErrorResponse(&w, herr)
			return
		}
		drainFunc = createdFunc
	} else {
		// Parse body as json to populate object
		obj, err = parseCreateObjectRequestAsJSON(r)
		// Validation
		if herr := handleCreatePrerequisites(ctx, h, &obj); herr != nil {
			sendAppErrorResponse(&w, herr)
			return
		}
	}
	obj.CreatedBy = caller.DistinguishedName
	// copy grant.EncryptKey to all existing permissions:
	for idx, permission := range obj.Permissions {
		permission.EncryptKey = grant.EncryptKey
		obj.Permissions[idx].EncryptKey = grant.EncryptKey
	}
	// Don't wipe out existing permissions, just add the new one!
	obj.Permissions = append(obj.Permissions, grant)

	createdObject, err = h.DAO.CreateObject(&obj)
	if err != nil {
		sendErrorResponse(&w, 500, err, "error storing object")
		return
	}
	// For requests where a stream was provided, only drain off into S3 once we have a record
	if isMultipart {
		if drainFunc != nil {
			go drainFunc()
		}
	}
	// Jsonified response
	w.Header().Set("Content-Type", "application/json")
	protocolObject := mapping.MapODObjectToObject(&createdObject)
	//Write a link back to the user so that it's possible to do an update on this object
	data, err := json.MarshalIndent(protocolObject, "", "  ")
	if err != nil {
		log.Printf("Error marshalling json data:%v", err)
	}
	w.Write(data)
	countOKResponse()
}

/* This is used by both createObject and createFolder to do common tasks against created objects
   Returns true if the request is now handled - which happens in the case of errors that terminate
   the http request
*/
func handleCreatePrerequisites(
	ctx context.Context,
	h AppServer,
	requestObject *models.ODObject,
) *AppError {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, nil, "Could not determine user")
	}

	// If JavaScript passes parentId as emptry string, set it to nil to satisfy
	// the DAO.
	if string(requestObject.ParentID) == "" {
		requestObject.ParentID = nil
	}

	// Check if parent defined
	if requestObject.ParentID == nil {
		// No parent set, but need to setup permission for the creator
		newPermission := models.ODObjectPermission{}
		newPermission.Grantee = caller.DistinguishedName
		newPermission.AllowCreate = true
		newPermission.AllowRead = true
		newPermission.AllowUpdate = true
		newPermission.AllowDelete = true
		newPermission.AllowShare = true
		requestObject.Permissions = append(requestObject.Permissions, newPermission)
	} else {
		// Parent is defined, retrieve existing parent object from the data store

		parentObject := models.ODObject{}
		parentObject.ID = requestObject.ParentID
		dbParentObject, err := h.DAO.GetObject(parentObject, false)
		if err != nil {
			return NewAppError(500, err, "Error retrieving parent object")
		}

		// Check if the user has permissions to create child objects under the
		// parent.
		//		Permission.grantee matches caller, and AllowCreate is true
		authorizedToCreate := false
		if len(dbParentObject.Permissions) > 0 {
			for _, permission := range dbParentObject.Permissions {
				if permission.Grantee == caller.DistinguishedName &&
					permission.AllowRead && permission.AllowCreate {
					authorizedToCreate = true
					break
				}
			}
		} else {
			log.Println("WARNING: No permissions on the object!")
		}
		if !authorizedToCreate {
			return NewAppError(403, nil, "Unauthorized")
		}

		// Make sure the object isn't deleted. To remove an object from the trash,
		// use removeObjectFromTrash call.
		if dbParentObject.IsDeleted {
			switch {
			case dbParentObject.IsExpunged:
				return NewAppError(410, err, "The object no longer exists.")
			case dbParentObject.IsAncestorDeleted && !dbParentObject.IsDeleted:
				return NewAppError(405, err, "Unallowed to create child objects under a deleted object.")
			case dbParentObject.IsDeleted:
				return NewAppError(405, err,
					"The object under which this object is being created is currently in the trash. Use removeObjectFromTrash to restore it first.",
				)
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
				newPermission.AllowShare = permission.AllowShare || isCreator
				requestObject.Permissions = append(requestObject.Permissions, newPermission)
			}
		}
	}

	// Disallow creating as deleted
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		return NewAppError(428, nil, "Creating object in a deleted state is not allowed")
	}

	// Validate ACM
	rawAcmString := requestObject.RawAcm.String
	// Make sure its parseable
	parsedACM, err := acm.NewACMFromRawACM(rawAcmString)
	if err != nil {
		return NewAppError(428, err, "ACM provided could not be parsed")
	}
	// Ensure user is allowed this acm
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, requestObject)
	if err != nil {
		return NewAppError(500, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, err, "Unauthorized")
	}
	// Map the parsed acm
	requestObject.ACM = mapping.MapACMToODObjectACM(&parsedACM)

	// Setup meta data...
	requestObject.CreatedBy = caller.DistinguishedName
	requestObject.OwnedBy.String = caller.DistinguishedName
	requestObject.OwnedBy.Valid = true
	requestObject.ACM.CreatedBy = caller.DistinguishedName

	return nil
}

func isMultipartFormData(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return false
	}
	d, _, err := mime.ParseMediaType(ct)
	if err != nil || d != "multipart/form-data" {
		return false
	}
	return true
}

func parseCreateObjectRequestAsJSON(r *http.Request) (models.ODObject, error) {
	var jsonObject protocol.CreateObjectRequest
	object := models.ODObject{}
	var err error

	if r.Header.Get("Content-Type") != "application/json" {
		err = fmt.Errorf("Content-Type is '%s', expected application/json", r.Header.Get("Content-Type"))
		return object, err
	}

	// Decode to JSON
	err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	if err != nil {
		return object, err
	}

	// Map to internal object type
	object, err = mapping.MapCreateObjectRequestToODObject(&jsonObject)
	if err != nil {
		return object, err
	}

	return object, nil
}

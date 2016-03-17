package server

import (
	//"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/metadata/models/acm"
)

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
		return &AppError{Code: 500, Err: nil, Msg: "Could not determine user"}
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
			return &AppError{
				Code: 500,
				Err:  err,
				Msg:  "Error retrieving parent object",
			}
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
			return &AppError{
				Code: 403,
				Err:  nil,
				Msg:  "Unauthorized",
			}
		}

		// Make sure the object isn't deleted. To remove an object from the trash,
		// use removeObjectFromTrash call.
		if dbParentObject.IsDeleted {
			switch {
			case dbParentObject.IsExpunged:
				return &AppError{
					Code: 410,
					Err:  err,
					Msg:  "The object no longer exists.",
				}
			case dbParentObject.IsAncestorDeleted && !dbParentObject.IsDeleted:
				return &AppError{
					Code: 405,
					Err:  err,
					Msg:  "Unallowed to create child objects under a deleted object.",
				}
			case dbParentObject.IsDeleted:
				return &AppError{
					Code: 405,
					Err:  err,
					Msg:  "The object under which this object is being created is currently in the trash. Use removeObjectFromTrash to restore it first.",
				}
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

	log.Printf("There are %d permissions being added..", len(requestObject.Permissions))

	// Disallow creating as deleted
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		return &AppError{
			Code: 428,
			Err:  nil,
			Msg:  "Creating object in a deleted state is not allowed",
		}
	}

	// Validate ACM
	rawAcmString := requestObject.RawAcm.String
	// Make sure its parseable
	parsedACM, err := acm.NewACMFromRawACM(rawAcmString)
	if err != nil {
		return &AppError{Code: 428, Err: err, Msg: "ACM provided could not be parsed"}
	}
	// Ensure user is allowed this acm
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, requestObject)
	if err != nil {
		return &AppError{500, err, "Error communicating with authorization service"}
	}
	if !hasAACAccess {
		return &AppError{403, err, "Unauthorized"}
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

// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(ctx context.Context, w http.ResponseWriter,
	r *http.Request) {
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	var obj models.ODObject
	var createdObject models.ODObject
	var acm models.ODACM
	var grant models.ODObjectPermission
	var err error

	if r.Method == "POST" {
		grant.Grantee = caller.DistinguishedName
		grant.AllowRead = true
		grant.AllowCreate = true
		grant.AllowUpdate = true
		grant.AllowDelete = true
		grant.AllowShare = true

		obj.CreatedBy = caller.DistinguishedName
		acm.CreatedBy = caller.DistinguishedName

		rName := utils.CreateRandomName()
		fileKey := utils.CreateKey()
		iv := utils.CreateIV()
		obj.ContentConnector.String = rName
		obj.EncryptIV = iv
		grant.EncryptKey = fileKey
		multipartReader, err := r.MultipartReader()
		if err != nil {
			h.sendErrorResponse(w, 400, err, "Unable to get mime multipart")
			return
		}
		herr, err := h.acceptObjectUpload(ctx, multipartReader, &obj, &grant, true)
		if herr != nil {
			h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
			return
		}
		obj.Permissions = make([]models.ODObjectPermission, 1)
		obj.Permissions = append(obj.Permissions, grant)

		createdObject, err = h.DAO.CreateObject(&obj)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "error storing object")
			return
		}
	}

	//TODO: json response rendering
	w.Header().Set("Content-Type", "application/json")
	protocolObject := mapping.MapODObjectToObject(&createdObject)
	//Write a link back to the user so that it's possible to do an update on this object
	data, err := json.MarshalIndent(protocolObject, "", "  ")
	if err != nil {
		log.Printf("Error marshalling json data:%v", err)
	}
	w.Write(data)
}

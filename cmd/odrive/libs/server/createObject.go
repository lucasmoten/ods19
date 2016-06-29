package server

import (
	//"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	var obj models.ODObject
	var createdObject models.ODObject
	var grant models.ODObjectPermission
	var err error
	var herr *AppError
	var drainFunc func()

	grant.Grantee = caller.DistinguishedName
	grant.AllowRead = true
	grant.AllowCreate = true
	grant.AllowUpdate = true
	grant.AllowDelete = true
	grant.AllowShare = true
	//Store the key *encrypted* ... not plain!
	models.SetEncryptKey(h.MasterKey, &grant)

	// Determine if this request is being made with a file stream or without.
	// When a filestream is provided, there is a different handler that parses
	// the multipart form data
	isMultipart := isMultipartFormData(r)
	if isMultipart {

		rName := utils.CreateRandomName()
		iv := utils.CreateIV()
		obj.ContentConnector.String = rName
		obj.ContentConnector.Valid = true
		obj.EncryptIV = iv

		multipartReader, err := r.MultipartReader()
		if err != nil {
			return NewAppError(400, err, "Unable to get mime multipart")
		}
		createdFunc, herr, err := h.acceptObjectUpload(ctx, multipartReader, &obj, &grant, true)
		if herr != nil {
			return herr
		}
		drainFunc = createdFunc
	} else {
		// Parse body as json to populate object
		obj, herr = parseCreateObjectRequestAsJSON(r)
		if herr != nil {
			return herr
		}
		// Validation
		if herr := handleCreatePrerequisites(ctx, h, &obj); herr != nil {
			return herr
		}
	}
	obj.CreatedBy = caller.DistinguishedName
	// copy grant.EncryptKey to all existing permissions:

	for idx, permission := range obj.Permissions {
		models.CopyEncryptKey(h.MasterKey, &grant, &permission)
		models.CopyEncryptKey(h.MasterKey, &grant, &obj.Permissions[idx])
	}

	createdObject, err = dao.CreateObject(&obj)
	if err != nil {
		if isMultipart && obj.ContentConnector.Valid {
			d := h.DrainProvider
			rName := FileId(obj.ContentConnector.String)
			uploadedName := NewFileName(rName, "uploaded")
			removeError := d.Files().Remove(d.Resolve(uploadedName))
			if removeError != nil {
				logger.Error("cannot remove orphaned file", zap.String("rname", string(rName)))
			}
		}
		return NewAppError(500, err, "error storing object")
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
		LoggerFromContext(ctx).Error(
			"marshal json",
			zap.String("err", err.Error()),
		)
	}
	w.Write(data)
	return nil
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
	dao := DAOFromContext(ctx)

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

	// Normalize Grantees for Permissions passed in request object
	for _, permission := range requestObject.Permissions {
		permission.Grantee = config.GetNormalizedDistinguishedName(permission.Grantee)
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
		dbParentObject, err := dao.GetObject(parentObject, false)
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
			LoggerFromContext(ctx).Warn("no permissions on object")
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

	// Ensure user is allowed this acm
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, requestObject)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, err, "Unauthorized")
	}

	// Setup meta data...
	requestObject.CreatedBy = caller.DistinguishedName
	requestObject.OwnedBy.String = caller.DistinguishedName
	requestObject.OwnedBy.Valid = true

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

func parseCreateObjectRequestAsJSON(r *http.Request) (models.ODObject, *AppError) {
	var jsonObject protocol.CreateObjectRequest
	object := models.ODObject{}
	var err error

	if r.Header.Get("Content-Type") != "application/json" {
		//BUG: return an AppError so that this gets *counted*
		err = fmt.Errorf("Content-Type is '%s', expected application/json", r.Header.Get("Content-Type"))
		return object, NewAppError(400, err, "expected Content-Type application/json")
	}

	// Decode to JSON
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return object, NewAppError(400, err, "Could not parse json object as a protocol.CreateObjectRequest")
	}

	// Map to internal object type
	object, err = mapping.MapCreateObjectRequestToODObject(&jsonObject)
	if err != nil {
		return object, NewAppError(400, err, "Could not map request to internal database type")
	}

	return object, nil
}

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

// createObject is a method handler on AppServer for createObject microservice operation.
func (h AppServer) createObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	logger := LoggerFromContext(ctx)

	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	var obj models.ODObject
	var createdObject models.ODObject
	var err error
	var herr *AppError
	var drainFunc func()

	ownerPermission := permissionWithOwnerDefaults(caller)
	models.SetEncryptKey(h.MasterKey, &ownerPermission)

	// NOTE: this bool is used far below to call drainFunc
	isMultipart := contentTypeIsMultipartFormData(r)
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
		createdFunc, herr, err := h.acceptObjectUpload(ctx, multipartReader, &obj, &ownerPermission, true)
		if herr != nil {
			return herr
		}
		drainFunc = createdFunc
	} else {
		// Check headers
		herr = validateCreateObjectHeaders(r)
		if herr != nil {
			return herr
		}

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

	// For all permissions, make sure we're using the flatened value
	herr = h.flattenGranteeOnAllPermissions(ctx, &obj)
	if herr != nil {
		return herr
	}

	// Make sure permissions passed in that are read access are put into the acm
	if herr := injectReadPermissionsIntoACM(ctx, &obj); herr != nil {
		return herr
	}
	// Flatten ACM, then Normalize Read Permissions against ACM f_share
	err = h.flattenACM(logger, &obj)
	if err != nil {
		return NewAppError(400, err, "ACM provided could not be flattened")
	}
	if herr := normalizeObjectReadPermissions(ctx, &obj); herr != nil {
		return herr
	}
	// Final access check against altered ACM
	hasAACAccess := false
	hasAACAccess, err = h.isUserAllowedForObjectACM(ctx, &obj)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, err, "Unauthorized")
	}

	// recalculate permission mac for owner permission
	ownerPermission.PermissionMAC = models.CalculatePermissionMAC(h.MasterKey, &ownerPermission)
	// copy ownerPermission.EncryptKey to all existing permissions:
	for idx, permission := range obj.Permissions {
		models.CopyEncryptKey(h.MasterKey, &ownerPermission, &permission)
		models.CopyEncryptKey(h.MasterKey, &ownerPermission, &obj.Permissions[idx])
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

// newOwnerPermission returns a default permission for the creator of an object.
func permissionWithOwnerDefaults(caller Caller) models.ODObjectPermission {
	var ownerPermission models.ODObjectPermission
	ownerPermission.Grantee = caller.DistinguishedName

	// Read permission not implicitly granted to owner. Must come through ACM share
	// (empty=everyone gets read, values=owner must be in one of those groups)
	ownerPermission.AllowRead = false
	ownerPermission.AllowCreate = true
	ownerPermission.AllowUpdate = true
	ownerPermission.AllowDelete = true
	ownerPermission.AllowShare = true
	return ownerPermission
}

// handleCreatePrerequisites used by both createObject and createFolder to do common tasks against created objects
// Returns true if the request is now handled - which happens in the case of errors that terminate
// the http request
func handleCreatePrerequisites(ctx context.Context, h AppServer, requestObject *models.ODObject) *AppError {
	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)

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
		addPermissionForCaller(ctx, requestObject)
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
		if ok := isUserAllowedToCreate(ctx, h.MasterKey, &dbParentObject); !ok {
			return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to create children under this object")
		}

		// Make sure the object isn't deleted.
		if dbParentObject.IsDeleted {
			switch {
			case dbParentObject.IsExpunged:
				return NewAppError(410, err, "object is expunged")
			case dbParentObject.IsAncestorDeleted:
				return NewAppError(405, err, "cannot create object under deleted anscestor")
			default:
				return NewAppError(405, err, "object is deleted")
			}
		}

		// Copy permissions from parent into request Object that are other than 'read only' which is tied to ACM f_share
		inheritPermissions := false // Disabled as it conflicts with ACM
		if inheritPermissions {
			for _, permission := range dbParentObject.Permissions {
				if !permission.IsDeleted && (permission.AllowCreate || permission.AllowUpdate || permission.AllowDelete || permission.AllowShare) {
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
		} else {
			addPermissionForCaller(ctx, requestObject)
		}
	}

	// Disallow creating as deleted
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		return NewAppError(428, nil, "Creating object in a deleted state is not allowed")
	}

	// Setup meta data...
	requestObject.CreatedBy = caller.DistinguishedName
	requestObject.OwnedBy.String = caller.DistinguishedName
	requestObject.OwnedBy.Valid = true

	return nil
}

func addPermissionForCaller(ctx context.Context, obj *models.ODObject) {
	caller, _ := CallerFromContext(ctx)
	newPermission := models.ODObjectPermission{}
	newPermission.Grantee = caller.DistinguishedName
	newPermission.AllowCreate = true
	// Read permission not implicitly granted to owner.
	// Must come through ACM share (empty=everyone gets read, values=owner must be in one of those groups)
	newPermission.AllowRead = false
	newPermission.AllowUpdate = true
	newPermission.AllowDelete = true
	newPermission.AllowShare = true
	obj.Permissions = append(obj.Permissions, newPermission)
}

func contentTypeIsMultipartFormData(r *http.Request) bool {
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

func validateCreateObjectHeaders(r *http.Request) *AppError {
	if r.Header.Get("Content-Type") != "application/json" {
		err := fmt.Errorf("Content-Type is '%s', expected application/json", r.Header.Get("Content-Type"))
		return NewAppError(400, err, "expected Content-Type application/json")
	}
	return nil
}

// injectReadPermissionsIntoACM iterates the permissions on an object, and for
// those granting read access, the share target equivalent of the grantee that
// is stored in AcmShare initialized when the permission was mapped, is then
// combined into the existing ACM. This function is ONLY intended for use when
// creating an object and passing permissions simultaneously that grant read
// access, and is used for preprocessing those permissions into the ACM before
// normalizing the permissions based upon the ACM.
func injectReadPermissionsIntoACM(ctx context.Context, obj *models.ODObject) *AppError {
	for i := len(obj.Permissions) - 1; i >= 0; i-- {
		permission := obj.Permissions[i]
		if permission.AllowRead && len(permission.AcmShare) > 0 {
			herr, sourceInterface := getACMInterfacePart(obj, "share")
			if herr != nil {
				return herr
			}
			interfaceToAdd, err := utils.UnmarshalStringToInterface(permission.AcmShare)
			if err != nil {
				return NewAppError(500, err, "Unable to unmarshal share from permission", zap.String("permission acmshare", permission.AcmShare))
			}
			combinedInterface := CombineInterface(sourceInterface, interfaceToAdd)
			herr = setACMPartFromInterface(ctx, obj, "share", combinedInterface)
			if herr != nil {
				return herr
			}
		}
	}
	return nil
}

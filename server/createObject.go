package server

import (
	//"encoding/hex"

	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/crypto"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/services/audit"
	"golang.org/x/net/context"

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

// createObject is a method handler on AppServer for createObject microservice operation.
func (h AppServer) createObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "create"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventCreate")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "CREATE")

	dao := DAOFromContext(ctx)

	var obj models.ODObject
	var createdObject models.ODObject
	var err error
	var herr *AppError
	var drainFunc func()

	// Only used for the encryptkey and assignment later. Actual owner permission set in handleCreatePrerequisites
	ownerPermission := permissionWithOwnerDefaults(caller)

	//After we parse the metadata, we need to set the encrypt key on the permission object
	afterMeta := func(obj *models.ODObject) {
		dp := ciphertext.FindCiphertextCacheByObject(obj)
		models.SetEncryptKey(dp.GetMasterKey(), &ownerPermission)
	}

	// NOTE: this bool is used far below to call drainFunc
	isMultipart := contentTypeIsMultipartFormData(r)
	if isMultipart {

		// Streamed objects have an IV
		iv := crypto.CreateIV()
		obj.EncryptIV = iv

		// Assign uniquely generated reference
		// NOTE: we could generate a software GUID here, and unify our object IDs.
		rName := crypto.CreateRandomName()
		obj.ContentConnector = models.ToNullString(rName)

		multipartReader, err := r.MultipartReader()
		if err != nil {
			herr := NewAppError(400, err, "Unable to get mime multipart")
			h.publishError(gem, herr)
			return herr
		}

		drainFunc, herr = h.acceptObjectUpload(ctx, multipartReader, &obj, &ownerPermission, true, afterMeta)
		if herr != nil {
			dp := ciphertext.FindCiphertextCacheByObject(&obj)
			h.publishError(gem, herr)
			return abortUploadObject(logger, dp, &obj, isMultipart, herr)
		}
	} else {
		// Check headers
		herr = validateCreateObjectHeaders(r)
		if herr != nil {
			h.publishError(gem, herr)
			return herr
		}

		// Parse body as json to populate object
		obj, herr = parseCreateObjectRequestAsJSON(r)
		if herr != nil {
			h.publishError(gem, herr)
			return herr
		}

		// Validation
		if herr := handleCreatePrerequisites(ctx, h, &obj); herr != nil {
			h.publishError(gem, herr)
			return herr
		}
	}
	obj.CreatedBy = caller.DistinguishedName

	dp := ciphertext.FindCiphertextCacheByObject(&obj)
	masterKey := dp.GetMasterKey()

	// Make sure permissions passed in that are read access are put into the acm
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	modifiedACM, err := aacAuth.InjectPermissionsIntoACM(obj.Permissions, obj.RawAcm.String)
	if err != nil {
		herr := NewAppError(500, err, "Error injecting provided permissions")
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}
	modifiedACM, err = aacAuth.GetFlattenedACM(modifiedACM)
	if err != nil {
		herr = ClassifyFlattenError(err)
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}
	obj.RawAcm = models.ToNullString(modifiedACM)
	modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(obj.OwnedBy.String, obj.Permissions, modifiedACM, obj.IsCreating())
	if err != nil {
		herr = NewAppError(500, err, err.Error())
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}
	obj.RawAcm = models.ToNullString(modifiedACM)
	obj.Permissions = modifiedPermissions
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, obj.RawAcm.String); err != nil {
		herr = ClassifyObjectACMError(err)
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}

	// recalculate permission mac for owner permission
	ownerPermission.PermissionMAC = models.CalculatePermissionMAC(masterKey, &ownerPermission)
	consolidateChangingPermissions(&obj)
	// copy ownerPermission.EncryptKey to all existing permissions:
	for idx, permission := range obj.Permissions {
		models.CopyEncryptKey(masterKey, &ownerPermission, &permission)
		models.CopyEncryptKey(masterKey, &ownerPermission, &obj.Permissions[idx])
	}

	user, _ := UserFromContext(ctx)
	snippetFields, _ := SnippetsFromContext(ctx)
	user.Snippets = snippetFields
	if err = handleIntermediateFoldersDuringCreation(dao, user, dp.GetMasterKey(), &obj); err != nil {
		herr = NewAppError(500, err, "error processing intermediate folders")
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}

	createdObject, err = dao.CreateObject(&obj)
	if err != nil {
		herr = NewAppError(500, err, "error storing object")
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}
	auditResource := NewResourceFromObject(createdObject)
	// For requests where a stream was provided, only drain off into S3 once we have a record,
	// and we pass all security checks.  Note that in between acceptObjectUpload and here,
	// we must call abortUploadObject to return early, so that we don't leave trash in the cache.
	if isMultipart {
		if drainFunc != nil {
			go drainFunc()
		}
	}

	apiResponse := mapping.MapODObjectToObject(&createdObject).WithCallerPermission(protocolCaller(caller))
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(createdObject.ID))
	gem.Payload.ObjectID = apiResponse.ID
	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.StreamUpdate = isMultipart
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, auditResource)

	jsonResponse(w, apiResponse)

	h.publishSuccess(gem, w)
	return nil
}

// handleIntermediateFoldersDuringCreation parses the object name and as necessary creates folder
// hierarchy leading up to the object name delimited by / and \ reserved characters. Note that this
// should not be applied for object updates since it can change the location of an object which is
// restricted as a separate operation, only permitted to owners.
func handleIntermediateFoldersDuringCreation(mdb dao.DAO, user models.ODUser, masterKey string, obj *models.ODObject) error {

	partName := trimPathDelimitersFromNameReturnAnyPart(obj)
	if partName != "" {
		// Get existing objects whose name matches this part
		matchedObjects, err := getObjectsWithName(mdb, user, obj.OwnedBy.String, partName, obj.ParentID)
		if err != nil {
			return err
		}

		// If an object already exists for this part ...
		var matchedObject models.ODObject
		if matchedObjects.TotalRows > 0 {
			matchedObject = matchedObjects.Objects[0]
		} else {
			// There is no object with this part name yet.  need to create it. Use same
			// settings for the acm and permissions
			folderObj := newFolderBasedOnObject(obj)
			folderObj.Name = partName
			// permissions need to be copied as new objects as they'll get ids assigned
			// during the create call
			for _, permission := range obj.Permissions {
				newPermission := newCopyOfPermission(permission, folderObj.CreatedBy)
				models.SetEncryptKey(masterKey, &newPermission)
				folderObj.Permissions = append(folderObj.Permissions, newPermission)
			}
			matchedObject, err = mdb.CreateObject(&folderObj)
			if err != nil {
				return err
			}
		}
		// Shift the parent id for the object being created
		obj.ParentID = matchedObject.ID

		return handleIntermediateFoldersDuringCreation(mdb, user, masterKey, obj)
	}
	return nil
}

func trimPathDelimitersFromNameReturnAnyPart(obj *models.ODObject) string {
	// trim leading and trailing slashes
	for strings.HasPrefix(obj.Name, "/") || strings.HasPrefix(obj.Name, "\\") {
		obj.Name = obj.Name[1:]
	}
	for strings.HasSuffix(obj.Name, "/") || strings.HasSuffix(obj.Name, "\\") {
		obj.Name = obj.Name[:len(obj.Name)-1]
	}

	// if there are slashes in the name, we need to change to location
	dpos := strings.IndexAny(obj.Name, "/\\")
	if dpos > -1 {
		partName := obj.Name[:dpos]
		obj.Name = obj.Name[dpos+1:]
		return partName
	}
	return ""
}

func getObjectsWithName(mdb dao.DAO, user models.ODUser, ownedBy string, name string, parentID []byte) (models.ODObjectResultset, error) {
	// Get existing objects whose name matches this part
	pagingRequest := dao.PagingRequest{FilterSettings: []dao.FilterSetting{dao.FilterSetting{FilterField: "name", Condition: "equals", Expression: name}}}
	var matchedObjects models.ODObjectResultset
	var err error
	if strings.HasPrefix(ownedBy, "user/") {
		if parentID == nil {
			matchedObjects, err = mdb.GetRootObjectsByUser(user, pagingRequest)
		} else {
			matchedObjects, err = mdb.GetChildObjectsByUser(user, pagingRequest, models.ODObject{ID: parentID})
		}
	} else {
		// TODO: when possible to create objects with group as owner, similar funcs will be called to match
	}
	return matchedObjects, err
}

func newFolderBasedOnObject(obj *models.ODObject) models.ODObject {
	theFolder := models.ODObject{
		CreatedBy:   obj.CreatedBy,
		OwnedBy:     obj.OwnedBy,
		TypeName:    models.ToNullString("Folder"),
		ParentID:    obj.ParentID,
		RawAcm:      obj.RawAcm,
		Permissions: []models.ODObjectPermission{},
	}
	return theFolder
}

func newCopyOfPermission(permission models.ODObjectPermission, createdBy string) models.ODObjectPermission {
	theCopy := models.ODObjectPermission{
		AcmGrantee: models.ODAcmGrantee{
			DisplayName:           permission.AcmGrantee.DisplayName,
			Grantee:               permission.AcmGrantee.Grantee,
			GroupName:             permission.AcmGrantee.GroupName,
			ProjectDisplayName:    permission.AcmGrantee.ProjectDisplayName,
			ProjectName:           permission.AcmGrantee.ProjectName,
			UserDistinguishedName: permission.AcmGrantee.UserDistinguishedName,
			ResourceString:        permission.AcmGrantee.ResourceString,
		},
		AcmShare:    permission.AcmShare,
		AllowCreate: permission.AllowCreate,
		AllowDelete: permission.AllowDelete,
		AllowRead:   permission.AllowRead,
		AllowShare:  permission.AllowShare,
		AllowUpdate: permission.AllowUpdate,
		CreatedBy:   createdBy,
		Grantee:     permission.Grantee,
	}
	return theCopy
}

// permissionWithOwnerDefaults returns a default permission for the creator of an object.
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
	if requestObject.ParentID != nil {
		// Parent is defined, retrieve existing parent object from the data store

		parentObject := models.ODObject{}
		parentObject.ID = requestObject.ParentID
		dbParentObject, err := dao.GetObject(parentObject, false)
		if err != nil {
			return NewAppError(500, err, "Error retrieving parent object")
		}

		// Check if the user has permissions to create child objects under the
		// parent.
		if ok := isUserAllowedToCreate(ctx, &dbParentObject); !ok {
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
	}

	// Disallow creating as deleted
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		return NewAppError(428, errors.New("Creating object in a deleted state is not allowed"), "Creating object in a deleted state is not allowed")
	}

	// Setup meta data...
	requestObject.CreatedBy = caller.DistinguishedName
	requestObject.OwnedBy = models.ToNullString("user/" + caller.DistinguishedName)

	// Give owner full CRUDS (read given by acm share)
	ownerCRUDS, _ := models.PermissionForOwner(requestObject.OwnedBy.String)
	ownerCUDS := models.PermissionWithoutRead(ownerCRUDS)
	requestObject.Permissions = append(requestObject.Permissions, ownerCUDS)

	return nil
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
		return object, NewAppError(400, err, "Could not map request to internal struct type")
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

func removeOrphanedFile(logger zap.Logger, d ciphertext.CiphertextCache, contentConnector string) {
	fileID := ciphertext.FileId(contentConnector)
	uploadedName := ciphertext.NewFileName(fileID, "uploaded")
	var err error
	if d != nil {
		err = d.Files().Remove(d.Resolve(uploadedName))
	}
	if err != nil {
		logger.Error("cannot remove orphaned file", zap.String("fileID", string(fileID)))
	}
}

func consolidateChangingPermissions(obj *models.ODObject) {
	var consolidated []models.ODObjectPermission
	for _, perm := range obj.Permissions {
		found := false
		if !perm.IsDeleted {
			for cidx, cPerm := range consolidated {
				if !cPerm.IsDeleted && cPerm.IsCreating() && perm.IsCreating() && (strings.Compare(cPerm.Grantee, perm.Grantee) == 0) {
					found = true
					cPerm.AllowCreate = cPerm.AllowCreate || perm.AllowCreate
					cPerm.AllowRead = cPerm.AllowRead || perm.AllowRead
					cPerm.AllowUpdate = cPerm.AllowUpdate || perm.AllowUpdate
					cPerm.AllowDelete = cPerm.AllowDelete || perm.AllowDelete
					cPerm.AllowShare = cPerm.AllowShare || perm.AllowShare
					consolidated[cidx] = cPerm
				}
			}
		}
		if !found {
			consolidated = append(consolidated, perm)
		}
	}
	obj.Permissions = consolidated
}

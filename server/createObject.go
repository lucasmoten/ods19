package server

import (
	//"encoding/hex"

	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/crypto"
	"github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/events"
	"github.com/deciphernow/object-drive-server/services/audit"
	"golang.org/x/net/context"

	"os"

	"github.com/deciphernow/object-drive-server/config"
	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/util"
)

// createObject creates an object or an object stream.
func (h AppServer) createObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)

	gem.Action = "create"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventCreate")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "CREATE")

	var obj models.ODObject
	var createdObject models.ODObject
	var pathDelimiter string
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

		iv := crypto.CreateIV()
		obj.EncryptIV = iv
		obj.ContentConnector = models.ToNullString(crypto.CreateRandomName())

		multipartReader, err := r.MultipartReader()
		if err != nil {
			herr := NewAppError(400, err, "Unable to get mime multipart")
			h.publishError(gem, herr)
			return herr
		}

		drainFunc, pathDelimiter, _, herr = h.acceptObjectUpload(ctx, multipartReader, &obj, &ownerPermission, true, afterMeta)
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
		obj, pathDelimiter, herr = parseCreateObjectRequestAsJSON(r)
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
		herr := NewAppError(authHTTPErr(err), err, "Error injecting provided permissions")
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}
	obj.RawAcm = models.ToNullString(modifiedACM)
	modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(obj.OwnedBy.String, obj.Permissions, modifiedACM, obj.IsCreating())
	if err != nil {
		herr = NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &obj, isMultipart, herr)
	}
	obj.RawAcm = models.ToNullString(modifiedACM)
	obj.Permissions = modifiedPermissions
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, obj.RawAcm.String); err != nil {
		herr = NewAppError(authHTTPErr(err), err, err.Error())
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
	if err = handleIntermediateFoldersDuringCreation(ctx, h, user, dp.GetMasterKey(), &obj, pathDelimiter); err != nil {
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
	parents, err := dao.GetParents(createdObject)
	if err != nil {
		herr := NewAppError(500, err, "error retrieving object parents")
		h.publishError(gem, herr)
		return herr
	}

	filtered := redactParents(ctx, aacAuth, parents)
	if appError := errOnDeletedParents(parents); appError != nil {
		h.publishError(gem, appError)
		return appError
	}
	crumbs := breadcrumbsFromParents(filtered)
	auditResource := NewResourceFromObject(createdObject)
	// For requests where a stream was provided, only drain off into S3 once we have a record,
	// and we pass all security checks.  Note that in between acceptObjectUpload and here,
	// we must call abortUploadObject to return early, so that we don't leave trash in the cache.
	if isMultipart {
		if drainFunc != nil {
			go drainFunc()
		}
	}

	apiResponse := mapping.MapODObjectToObject(&createdObject).WithCallerPermission(protocolCaller(caller)).WithBreadcrumbs(crumbs)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(createdObject.ID))
	gem.Payload.ObjectID = apiResponse.ID
	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.StreamUpdate = isMultipart
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, auditResource)
	gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
	jsonResponse(w, apiResponse)

	h.publishSuccess(gem, w)
	return nil
}

// handleIntermediateFoldersDuringCreation parses the object name and as necessary creates folder
// hierarchy leading up to the object name delimited by / and \ reserved characters. Note that this
// should not be applied for object updates since it can change the location of an object which is
// restricted as a separate operation, only permitted to owners.
func handleIntermediateFoldersDuringCreation(ctx context.Context, h AppServer, user models.ODUser, masterKey string, obj *models.ODObject, pathDelimiter string) error {

	gem, _ := GEMFromContext(ctx)
	gem.Action = "create"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventCreate")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "CREATE")
	gem = ResetBulkItem(gem)
	dao := DAOFromContext(ctx)

	// Determine actual path delimiter to use.
	if len(pathDelimiter) == 0 {
		pathDelimiter = util.DefaultPathDelimiter
	}

	partName := trimPathDelimitersFromNameReturnAnyPart(obj, pathDelimiter)
	if partName != "" {
		// Get existing objects whose name matches this part
		matchedObjects, err := getObjectsWithName(dao, user, obj.OwnedBy.String, partName, obj.ParentID)
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
			matchedObject, err = dao.CreateObject(&folderObj)
			if err != nil {
				return err
			}
			auditResource := NewResourceFromObject(matchedObject)
			gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(matchedObject.ID))
			gem.Payload.ObjectID = hex.EncodeToString(matchedObject.ID)
			gem.Payload.ChangeToken = matchedObject.ChangeToken
			gem.Payload.StreamUpdate = false
			gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, auditResource)
			gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "SUCCESS")
			gem.Payload.Audit = audit.WithActionTargetMessages(gem.Payload.Audit, string(http.StatusOK))
			apiResponse := mapping.MapODObjectToObject(&matchedObject)
			gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
			h.EventQueue.Publish(gem)
		}
		// Shift the parent id for the object being created
		obj.ParentID = matchedObject.ID

		return handleIntermediateFoldersDuringCreation(ctx, h, user, masterKey, obj, pathDelimiter)
	}
	return nil
}

func trimPathDelimitersFromNameReturnAnyPart(obj *models.ODObject, pathDelimiter string) string {
	part, remainder := util.GetNextDelimitedPart(obj.Name, pathDelimiter)
	obj.Name = remainder
	return part
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
	} else if strings.HasPrefix(ownedBy, "group/") {
		if parentID == nil {
			targetGrantee := models.NewODAcmGranteeFromResourceName(ownedBy)
			groupName := targetGrantee.Grantee
			matchedObjects, err = mdb.GetRootObjectsByGroup(groupName, user, pagingRequest)
		} else {
			matchedObjects, err = mdb.GetChildObjectsByUser(user, pagingRequest, models.ODObject{ID: parentID})
		}
	} else {
		// Resource type unsupported
		err = fmt.Errorf("Unable to get objects based upon ownedBy resource : %s", ownedBy)
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

// handleCreatePrerequisites used by both createObject and createFolder to do common tasks against created objects.
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

	requestObject.CreatedBy = caller.DistinguishedName
	// There may already be an ownedBy that is not the same as createdBy
	self := "user/" + requestObject.CreatedBy
	targetGroup := requestObject.OwnedBy.String
	if len(targetGroup) == 0 {
		requestObject.OwnedBy = models.ToNullString(self)
	} else {
		// owner is already set, so determine if group name causes it to get rejected
		// it can also be rejected if it's just another user.
		isAllowed := false
		// Allow us to set to ourselves.
		if strings.Compare(self, targetGroup) == 0 {
			isAllowed = true
		}
		// Otherwise, we must set to a group (which we "are" in some sense)
		if !isAllowed && strings.HasPrefix(targetGroup, "group/") {
			groups, ok := GroupsFromContext(ctx)
			if !ok {
				return NewAppError(500, errors.New("Error getting groups"), "Error getting groups")
			}
			if strings.Compare(strings.ToLower(targetGroup), strings.ToLower("group/-Everyone")) == 0 {
				return NewAppError(428, errors.New("Cannot assign to everyone group"), "Cannot assign to everyone group")
			}
			tg := models.NewODAcmGranteeFromResourceName(targetGroup)
			for _, g := range groups {
				if strings.Compare(g, tg.Grantee) == 0 {
					isAllowed = true
					// Apply normalized form
					requestObject.OwnedBy = models.ToNullString(tg.String())
					// No need to check every group the user has
					break
				}
			}
		}
		if !isAllowed {
			msg := "User must be in group being set as the owner"
			return NewAppError(428, errors.New(msg), msg)
		}
	}

	// Give owner full CRUDS (read given by acm share)
	// TODO(cm): this needs clarification
	ownerCRUDS, _ := models.PermissionForOwner(requestObject.OwnedBy.String)
	ownerCUDS := models.PermissionWithoutRead(ownerCRUDS)
	requestObject.Permissions = append(requestObject.Permissions, ownerCUDS)

	return nil
}

func contentTypeIsMultipartFormData(r *http.Request) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "multipart/form-data" {
		return false
	}
	return true
}

func parseCreateObjectRequestAsJSON(r *http.Request) (models.ODObject, string, *AppError) {
	var jsonObject protocol.CreateObjectRequest
	object := models.ODObject{}
	var err error

	// Decode to JSON
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return object, "", NewAppError(400, err, "Could not parse json object as a protocol.CreateObjectRequest")
	}

	// Map to internal object type
	err = mapping.OverwriteODObjectWithCreateObjectRequest(&object, &jsonObject)
	if err != nil {
		msg := fmt.Sprintf("Could not map request to internal struct type. %s", err.Error())
		return object, "", NewAppError(400, err, msg)
	}

	return object, jsonObject.NamePathDelimiter, nil
}

func validateCreateObjectHeaders(r *http.Request) *AppError {
	if !util.IsApplicationJSON(r.Header.Get("Content-Type")) {
		err := fmt.Errorf("Content-Type is '%s', expected application/json", r.Header.Get("Content-Type"))
		return NewAppError(400, err, "expected Content-Type application/json")
	}
	return nil
}

func removeOrphanedFile(logger *zap.Logger, d ciphertext.CiphertextCache, contentConnector string) {
	fileID := ciphertext.FileId(contentConnector)
	uploadedName := ciphertext.NewFileName(fileID, ".uploaded")
	orphanedName := ciphertext.NewFileName(fileID, ".orphaned")
	var err error
	if _, err := d.Files().Stat(d.Resolve(uploadedName)); os.IsNotExist(err) {
		logger.Info("file sent was not stored locally, no need to remove or rename")
		return
	}
	if d != nil {
		err = d.Files().Remove(d.Resolve(uploadedName))
	}
	if err != nil {
		logger.Error("cannot remove orphaned file. will attempt rename", zap.String("fileID", string(fileID)), zap.Error(err))
		err = d.Files().Rename(d.Resolve(uploadedName), d.Resolve(orphanedName))
		if err != nil {
			logger.Error("cannot rename uploaded file to orphaned state. check directory permissions in cache folder", zap.String("fileID", string(fileID)), zap.Error(err))
		}
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

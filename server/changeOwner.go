package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/util"

	"golang.org/x/net/context"
)

func (h AppServer) changeOwner(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error
	var recursive bool

	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "OWNERSHIP_MODIFY")
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	captured, _ := CaptureGroupsFromContext(ctx)

	requestObject, recursive, err = parseChangeOwnerRequestAsJSON(r, captured["objectId"], captured["newOwner"])
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "bad request")
		h.publishError(gem, herr)
		return herr
	}

	newOwnerStr := captured["newOwner"]

	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)

	// Auth checks
	okToUpdate, updatePermission := isUserAllowedToUpdateWithPermission(ctx, &dbObject)
	if !okToUpdate {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
		h.publishError(gem, herr)
		return herr
	}
	// Verify caller owns the object being changed or is member of group having ownership
	userGroupResourceStrings := getKnownResourceStringsFromUserGroups(ctx)
	if len(userGroupResourceStrings) == 0 {
		herr := NewAppError(http.StatusInternalServerError, errors.New("Forbidden"), "Forbidden - Server cannot load resource strings from user groups")
		h.publishError(gem, herr)
		return herr
	}
	if !aacAuth.IsUserOwner(caller.DistinguishedName, userGroupResourceStrings, dbObject.OwnedBy.String) {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User must be an object owner to transfer ownership of the object")
		h.publishError(gem, herr)
		return herr
	}
	// Verify target is either a different user (no way to validate yet), or a group the caller is a member of, and normalize the resource string
	newOwnerAcmGrantee := models.NewODAcmGranteeFromResourceName(requestObject.OwnedBy.String)
	targetResourceString := newOwnerAcmGrantee.ResourceName()
	if len(newOwnerAcmGrantee.UserDistinguishedName.String) > 0 {
		if newOwnerAcmGrantee.UserDistinguishedName.String == caller.DistinguishedName {
			msg := "Unable to change owner of object to self."
			herr := NewAppError(http.StatusBadRequest, errors.New(msg), msg)
			h.publishError(gem, herr)
			return herr
		}
		requestObject.OwnedBy = models.ToNullString(targetResourceString)
	} else if len(newOwnerAcmGrantee.GroupName.String) > 0 {
		if targetResourceString == "group/-everyone" {
			msg := "Cannot assign ownership of object to everyone group"
			herr := NewAppError(http.StatusPreconditionRequired, errors.New(msg), msg)
			h.publishError(gem, herr)
			return herr
		}
		allowed := false
		for _, groupString := range userGroupResourceStrings {
			log.Println(fmt.Sprintf("target: %s, groupstring: %s", targetResourceString, groupString))
			if groupString == targetResourceString {
				allowed = true
				requestObject.OwnedBy = models.ToNullString(targetResourceString)
				break
			}
		}
		if !allowed {
			msg := "User must be in group being set as the owner"
			herr := NewAppError(http.StatusPreconditionRequired, errors.New(msg), msg)
			h.publishError(gem, herr)
			return herr
		}
	} else {
		msg := fmt.Sprintf("Unrecognized value for new owner %s", requestObject.OwnedBy.String)
		herr := NewAppError(http.StatusBadRequest, errors.New(msg), msg)
		h.publishError(gem, herr)
		return herr
	}

	// Capture and overwrite here for comparison later after the update
	requestObject.ChangeCount = dbObject.ChangeCount
	apiResponse, herr := changeOwnerRaw(&requestObject, &dbObject, &updatePermission, aacAuth, caller, dao)
	if herr != nil {
		h.publishError(gem, herr)
		return herr
	}
	auditModified := NewResourceFromObject(dbObject)

	// Event broadcast
	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))

	jsonResponse(w, *apiResponse)
	h.publishSuccess(gem, w)

	// begin recursive application
	if recursive {
		go h.changeOwnerRecursive(ctx, newOwnerStr, requestObject.ID)
	}
	return nil
}

func (h AppServer) changeOwnerRecursive(ctx context.Context, newOwner string, id []byte) {
	d := DAOFromContext(ctx)
	rs := getKnownResourceStringsFromUserGroups(ctx)
	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "OWNERSHIP_MODIFY")

	page := 1
	pr := dao.PagingRequest{PageNumber: page, PageSize: dao.MaxPageSize}
	obj := models.ODObject{ID: id}

	children, err := d.GetChildObjectsWithProperties(pr, obj)
	if err != nil {
		logger.Error("error calling GetChildObjectsWithProperties", zap.Object("err", err))
		return
	}

	for {
		for _, child := range children.Objects {

			gem.Payload.ObjectID = hex.EncodeToString(child.ID)
			gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(child.ID))
			auditOriginal := NewResourceFromObject(child)

			if !aacAuth.IsUserOwner(caller.DistinguishedName, rs, child.OwnedBy.String) {
				continue
			}
			if child.IsDeleted {
				continue
			}

			perm, err := models.CreateODPermissionFromResource(newOwner)
			if err != nil {
				logger.Error("could not create new owner permission", zap.Object("err", err))
				gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
				h.EventQueue.Publish(gem)
				continue
			}

			if perm.AcmGrantee.Grantee == "" {
				logger.Error("grantee cannot be empty string", zap.Object("err", err))
				gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
				h.EventQueue.Publish(gem)
				continue
			}
			// Don't allow transfer to everyone
			if isPermissionFor(&perm, models.EveryoneGroup) {
				err = errors.New("cannot transfer ownership to everyone")
				logger.Error("error changing owner recursively", zap.Object("err", err))
				gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
				h.EventQueue.Publish(gem)
				continue
			}

			ok, existingPerm := isUserAllowedToUpdateWithPermission(ctx, &child)
			if !ok {
				logger.Error("grantee cannot be empty string", zap.Object("err", errors.New("caller cannot update object")))
				continue
			}
			// Owner gets full cruds
			perm.AllowCreate, perm.AllowRead, perm.AllowUpdate, perm.AllowDelete, perm.AllowShare = true, true, true, true, true
			masterKey := ciphertext.FindCiphertextCacheByObject(&child).GetMasterKey()
			models.CopyEncryptKey(masterKey, &existingPerm, &perm)
			child.Permissions = append(child.Permissions, perm)

			modifiedACM, err := aacAuth.InjectPermissionsIntoACM(child.Permissions, child.RawAcm.String)
			if err != nil {
				logger.Error("cannot inject permissions into child object", zap.Object("err", err))
				continue
			}
			modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(child.OwnedBy.String, child.Permissions, modifiedACM, false)
			if err != nil {
				logger.Error("error calling NormalizePermissionsFromACM", zap.Object("err", err))
				continue
			}
			child.RawAcm = models.ToNullString(modifiedACM)
			child.Permissions = modifiedPermissions

			// NOTE(cm): why the check again?
			if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, child.RawAcm.String); err != nil {
				logger.Error("error calling IsUserAuthorizedForACM", zap.Object("err", err))
				continue
			}
			consolidateChangingPermissions(&child)
			for i, p := range child.Permissions {
				models.CopyEncryptKey(masterKey, &existingPerm, &p)
				models.CopyEncryptKey(masterKey, &existingPerm, &child.Permissions[i])
			}
			child.ModifiedBy = caller.DistinguishedName
			// TODO(cm) move up earlier in this function?
			child.OwnedBy = models.ToNullString(newOwner)
			err = d.UpdateObject(&child)
			if err != nil {
				logger.Error("error updating child object with new permissions", zap.Object("err", err))
				continue
			}

			auditModified := NewResourceFromObject(child)
			gem.Payload.Audit = audit.WithModifiedPairList(
				gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
			h.EventQueue.Publish(gem)
			h.changeOwnerRecursive(ctx, newOwner, child.ID)

		}
		page++
		if page > children.PageCount {
			break
		}
		pr.PageNumber = page
		var err error
		children, err = d.GetChildObjectsWithProperties(pr, obj)
		if err != nil {
			logger.Error("error calling GetChildObjectsWithProperties", zap.Object("err", err))
			return
		}
	}
}

func changeOwnerRaw(
	requestObject, dbObject *models.ODObject,
	updatePermission *models.ODObjectPermission,
	aacAuth *auth.AACAuth,
	caller Caller,
	dao dao.DAO,
) (*protocol.Object, *AppError) {
	var err error
	// Object state check
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return nil, NewAppError(http.StatusGone, err, "object no longer exists")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return nil, NewAppError(http.StatusMethodNotAllowed, err, "object cannot be modified because an ancestor is deleted")
		case dbObject.IsDeleted:
			return nil, NewAppError(http.StatusMethodNotAllowed, err, "object is already in the trash")
		}
	}

	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		return nil, NewAppError(http.StatusPreconditionRequired, errors.New("ChangeToken does not match expected value"), "ChangeToken does not match expected value")
	}

	// Check that the owner of the object passed in is different then the current
	// state of the object in the data store
	if requestObject.OwnedBy.String == dbObject.OwnedBy.String {
		// NOOP, will return current state
		requestObject = dbObject
	} else {
		// Changing owner...

		// Parse from resource to permission with acmgrnatee
		newOwnerPermission, err := models.CreateODPermissionFromResource(requestObject.OwnedBy.String)
		// Validate that we were able to parse
		if err != nil {
			return nil, NewAppError(http.StatusBadRequest, err, err.Error())
		}
		// TODO(cm): Why this additional check? CreateODPermissionsFromResource should return err.
		if newOwnerPermission.AcmGrantee.Grantee == "" {
			msg := "Value provided for new owner could not be parsed"
			err = fmt.Errorf("%s: %s", msg, requestObject.OwnedBy.String)
			return nil, NewAppError(http.StatusBadRequest, err, msg)
		}
		// Don't allow transferring to everyone
		if isPermissionFor(&newOwnerPermission, models.EveryoneGroup) {
			err = errors.New("Transferring ownership to everyone is not allowed")
			return nil, NewAppError(http.StatusBadRequest, err, err.Error())
		}

		// Move the top-level changed object to the new owner's root. If child objects
		// are affected recursively, they should become children under this top-level object.
		dbObject.ParentID = nil

		// New owner gets full CRUDS automatically.
		newOwnerPermission.AllowCreate = true
		newOwnerPermission.AllowRead = true
		newOwnerPermission.AllowUpdate = true
		newOwnerPermission.AllowDelete = true
		newOwnerPermission.AllowShare = true
		masterKey := ciphertext.FindCiphertextCacheByObject(dbObject).GetMasterKey()
		models.CopyEncryptKey(masterKey, updatePermission, &newOwnerPermission)
		dbObject.Permissions = append(dbObject.Permissions, newOwnerPermission)

		// Inject into ACM and Rebuild
		modifiedACM, err := aacAuth.InjectPermissionsIntoACM(dbObject.Permissions, dbObject.RawAcm.String)
		if err != nil {
			return nil, NewAppError(authHTTPErr(err), err, "Error injecting permission for new owner")
		}
		modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(dbObject.OwnedBy.String, dbObject.Permissions, modifiedACM, dbObject.IsCreating())
		if err != nil {
			return nil, NewAppError(authHTTPErr(err), err, err.Error())
		}
		dbObject.RawAcm = models.ToNullString(modifiedACM)
		dbObject.Permissions = modifiedPermissions
		if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
			return nil, NewAppError(authHTTPErr(err), err, err.Error())
		}

		consolidateChangingPermissions(dbObject)

		for idx, permission := range dbObject.Permissions {
			models.CopyEncryptKey(masterKey, updatePermission, &permission)
			models.CopyEncryptKey(masterKey, updatePermission, &dbObject.Permissions[idx])
		}

		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.OwnedBy = requestObject.OwnedBy
		err = dao.UpdateObject(dbObject)
		if err != nil {
			log.Printf("Error updating object: %v", err)
			return nil, NewAppError(http.StatusInternalServerError, nil, "Error saving object with new owner")
		}

		// After the update, check that key values have changed...
		if dbObject.ChangeCount <= requestObject.ChangeCount {
			return nil, NewAppError(http.StatusInternalServerError, nil, "ChangeCount didn't update when processing owner transfer request")
		}
		if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
			return nil, NewAppError(http.StatusInternalServerError, nil, "ChangeToken didn't update when processing owner transfer request")
		}
	}

	apiResponse := mapping.MapODObjectToObject(dbObject).WithCallerPermission(protocolCaller(caller))

	return &apiResponse, nil
}

func parseChangeOwnerRequestAsJSON(r *http.Request, objectID string, newOwner string) (models.ODObject, bool, error) {
	var jsonObject protocol.ChangeOwnerRequest
	var requestObject models.ODObject
	var err error

	if !util.IsApplicationJSON(r.Header.Get("Content-Type")) {
		return requestObject, false, errors.New("expected header Content-Type: application/json")
	}

	// Depends on this for the changeToken
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return requestObject, false, err
	}

	// Initialize requestobject with the objectId being requested
	if objectID == "" {
		return requestObject, false, errors.New("could not extract ObjectID from URI")
	}
	_, err = hex.DecodeString(objectID)
	if err != nil {
		return requestObject, false, errors.New("invalid ObjectID in URI")
	}
	jsonObject.ID = objectID
	// And the new owner
	if len(newOwner) > 0 {
		jsonObject.NewOwner = newOwner
	} else {
		return requestObject, false, errors.New("A new owner is required when changing owner")
	}

	// Map to internal object type
	requestObject, err = mapping.MapChangeOwnerRequestToODObject(&jsonObject)
	return requestObject, jsonObject.ApplyRecursively, err
}

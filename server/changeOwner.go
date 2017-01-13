package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

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

	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "OWNERSHIP_MODIFY")
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	captured, _ := CaptureGroupsFromContext(ctx)
	// Get object
	if r.Header.Get("Content-Type") != "application/json" {
		herr := NewAppError(http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		h.publishError(gem, herr)
		return herr
	}
	requestObject, err = parseChangeOwnerRequestAsJSON(r, captured["objectId"], captured["newOwner"])
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)

	// Auth check
	okToUpdate, updatePermission := isUserAllowedToUpdateWithPermission(ctx, &dbObject)
	if !okToUpdate {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
		h.publishError(gem, herr)
		return herr
	}
	if !aacAuth.IsUserOwner(caller.DistinguishedName, getKnownResourceStringsFromUserGroups(ctx), dbObject.OwnedBy.String) {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User must be an object owner to transfer ownership of the object")
		h.publishError(gem, herr)
		return herr
	}

	// Capture and overwrite here for comparison later after the update
	requestObject.ChangeCount = dbObject.ChangeCount
	apiResponse, herr := changeOwnerRaw(
		&requestObject,
		&dbObject,
		&updatePermission,
		aacAuth,
		caller,
		dao,
	)
	if herr != nil {
		h.publishError(gem, herr)
		return herr
	}
	auditModified := NewResourceFromObject(dbObject)

	// Event broadcast
	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
	h.publishSuccess(gem, r)

	jsonResponse(w, *apiResponse)
	return nil
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
			return nil, NewAppError(http.StatusGone, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return nil, NewAppError(http.StatusMethodNotAllowed, err, "The object cannot be modified because an ancestor is deleted.")
		case dbObject.IsDeleted:
			return nil, NewAppError(http.StatusMethodNotAllowed, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
		}
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		return nil, NewAppError(http.StatusPreconditionRequired, errors.New("Precondition required: ChangeToken does not match expected value"), "ChangeToken does not match expected value. Object may have been changed by another request.")
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
		if newOwnerPermission.AcmGrantee.Grantee == "" {
			msg := "Value provided for new owner could not be parsed"
			err = fmt.Errorf("%s: %s", msg, requestObject.OwnedBy.String)
			return nil, NewAppError(http.StatusBadRequest, err, msg)
		}
		// Don't allow transferring to everyone
		if isPermissionFor(&newOwnerPermission, models.EveryoneGroup) {
			msg := "Transferring ownership to everyone is not allowed"
			err = fmt.Errorf("%s", msg)
			return nil, NewAppError(http.StatusBadRequest, err, msg)
		}

		// Force to root
		dbObject.ParentID = nil

		// Setup cruds
		dp := ciphertext.FindCiphertextCacheByObject(dbObject)
		masterKey := dp.GetMasterKey()
		newOwnerPermission.AllowCreate = true
		newOwnerPermission.AllowRead = true
		newOwnerPermission.AllowUpdate = true
		newOwnerPermission.AllowDelete = true
		newOwnerPermission.AllowShare = true
		models.CopyEncryptKey(masterKey, updatePermission, &newOwnerPermission)
		dbObject.Permissions = append(dbObject.Permissions, newOwnerPermission)

		// Inject into ACM and Rebuild
		modifiedACM, err := aacAuth.InjectPermissionsIntoACM(dbObject.Permissions, dbObject.RawAcm.String)
		if err != nil {
			return nil, NewAppError(500, err, "Error injecting permission for new owner")
		}
		modifiedACM, err = aacAuth.GetFlattenedACM(modifiedACM)
		if err != nil {
			return nil, ClassifyFlattenError(err)
		}
		dbObject.RawAcm = models.ToNullString(modifiedACM)
		modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(dbObject.OwnedBy.String, dbObject.Permissions, modifiedACM, dbObject.IsCreating())
		if err != nil {
			return nil, NewAppError(500, err, err.Error())
		}
		dbObject.RawAcm = models.ToNullString(modifiedACM)
		dbObject.Permissions = modifiedPermissions
		if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
			return nil, ClassifyObjectACMError(err)
		}

		// Consolidate permissions
		consolidateChangingPermissions(dbObject)
		// copy ownerPermission.EncryptKey to all existing permissions:
		for idx, permission := range dbObject.Permissions {
			models.CopyEncryptKey(masterKey, updatePermission, &permission)
			models.CopyEncryptKey(masterKey, updatePermission, &dbObject.Permissions[idx])
		}

		// Call metadata connector to update the object in the data store
		// We reference the dbObject here instead of request to isolate what is
		// allowed to be changed in this operation
		// Force the modified by to be that of the caller
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

func parseChangeOwnerRequestAsJSON(r *http.Request, objectID string, newOwner string) (models.ODObject, error) {
	var jsonObject protocol.ChangeOwnerRequest
	var requestObject models.ODObject
	var err error

	// Depends on this for the changeToken
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return requestObject, err
	}

	// Initialize requestobject with the objectId being requested
	if objectID == "" {
		return requestObject, errors.New("Could not extract ObjectID from URI")
	}
	_, err = hex.DecodeString(objectID)
	if err != nil {
		return requestObject, errors.New("Invalid ObjectID in URI")
	}
	jsonObject.ID = objectID
	// And the new owner
	if len(newOwner) > 0 {
		jsonObject.NewOwner = newOwner
	} else {
		return requestObject, errors.New("A new owner is required when changing owner")
	}

	// Map to internal object type
	requestObject, err = mapping.MapChangeOwnerRequestToODObject(&jsonObject)
	return requestObject, err
}

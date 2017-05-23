package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) addObjectShare(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	// TODO(cm): deprecate this handler.
	caller, _ := CallerFromContext(ctx)
	logger := LoggerFromContext(ctx)
	dao := DAOFromContext(ctx)
	session := SessionIDFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	var err error

	rollupPermission, permissions, dbObject, herr := commonObjectSharePrep(ctx, r)
	if herr != nil {
		return herr
	}

	dp := ciphertext.FindCiphertextCacheByObject(&dbObject)
	masterKey := dp.GetMasterKey()

	// Only proceed if there were permissions provided
	if len(permissions) == 0 {
		logger.Info("No permissions derived from share for adding.")
	} else {
		permissionsChanged := false

		// Check if database object has a read permission for everyone
		dbHasEveryone := hasPermissionsForGrantee(&dbObject, models.EveryoneGroup)

		// Check if removing everyone
		removingEveryone := false
		if dbHasEveryone {
			for _, permission := range permissions {
				if permission.AllowRead && !isPermissionFor(&permission, models.EveryoneGroup) {
					removePermissionsForGrantee(&dbObject, models.EveryoneGroup)
					// since removed everyone, reset the flag so we dont do this check for every permission
					dbHasEveryone = false
					removingEveryone = true
					permissionsChanged = true
				}
			}
		}

		// Force Owner CRUDS
		if removingEveryone {
			_, ownerR := models.PermissionForOwner(dbObject.OwnedBy.String)
			permissions = append(permissions, ownerR)
		}

		// Iterate the permissions, condensing to only new being added
		for _, permission := range permissions {

			// Verify that permission settings are allowed for user's rollupPermission
			if herr := verifyPermissionToShare(rollupPermission, permission, false); herr != nil {
				return herr
			}

			// Metadata for this permission to be created
			permission.ObjectID = dbObject.ID
			permission.CreatedBy = caller.DistinguishedName
			permission.ExplicitShare = true

			// If after removing existing grants there are no more permissions, ...
			if reduceGrantsFromExistingPermissionsLeavesNone(dbObject.Permissions, &permission) {
				// stop processing this permission
				continue
			}

			// Add to list of permissions being added
			permissionsChanged = true
			models.CopyEncryptKey(masterKey, &rollupPermission, &permission)
			dbObject.Permissions = append(dbObject.Permissions, permission)
		}

		// If actual changes from removing everyone or adding capabilities...
		if permissionsChanged {
			dbObject.ModifiedBy = caller.DistinguishedName
			// Post modification authorization checks
			aacAuth := auth.NewAACAuth(logger, h.AAC)
			modifiedACM := dbObject.RawAcm.String
			// Rebuild
			modifiedACM, err = aacAuth.RebuildACMFromPermissions(dbObject.Permissions, modifiedACM)
			if err != nil {
				return NewAppError(500, err, "Error rebuilding ACM from revised permissions")
			}
			// Flatten
			var msgs []string
			modifiedACM, msgs, err = aacAuth.GetFlattenedACM(modifiedACM)
			if err != nil {
				return NewAppError(authHTTPErr(err), err, err.Error()+strings.Join(msgs, "/"))
			}
			dbObject.RawAcm = models.ToNullString(modifiedACM)
			// Check that caller has access
			if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
				return NewAppError(authHTTPErr(err), err, err.Error())
			}

			// Finally update the object in database, which handles permissions transactionally
			if err := dao.UpdateObject(&dbObject); err != nil {
				return NewAppError(500, err, "Error updating object")
			}
		}
	}

	// Now fetch updated object
	updatedObject, err := dao.GetObject(dbObject, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	apiResponse := mapping.MapODObjectToObject(&updatedObject).WithCallerPermission(protocolCaller(caller))

	gem.Action = "update"
	gem.Payload = events.ObjectDriveEvent{
		ObjectID:     apiResponse.ID,
		ChangeToken:  apiResponse.ChangeToken,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: false,
		SessionID:    session,
	}
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PERMISSION_MODIFY")
	gem.Payload.ObjectID = hex.EncodeToString(updatedObject.ID)
	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(updatedObject.ID))
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(updatedObject))
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)
	return nil
}

func commonObjectSharePrep(ctx context.Context, r *http.Request) (models.ODObjectPermission, []models.ODObjectPermission, models.ODObject, *AppError) {

	// Get dao value from ctx.
	dao := DAOFromContext(ctx)

	var rollupPermission models.ODObjectPermission
	var permissions []models.ODObjectPermission
	var dbObject models.ODObject

	// Get the object ID from the request
	bytesObjectID, err := getObjectIDFromContext(ctx)
	if err != nil {
		return rollupPermission, permissions, dbObject, NewAppError(400, err, err.Error())
	}

	// Load the existing object
	requestedObject := models.ODObject{}
	requestedObject.ID = bytesObjectID
	dbObject, err = dao.GetObject(requestedObject, false)
	if err != nil {
		return rollupPermission, permissions, dbObject, NewAppError(500, err, "Error retrieving object")
	}

	// Check Permissions
	allowedToShare := false
	allowedToShare, rollupPermission = isUserAllowedToShareWithPermission(ctx, &dbObject)
	if !allowedToShare {
		return rollupPermission, permissions, dbObject, NewAppError(403, errors.New("unauthorized to share"), "Forbidden - User does not have permission to modify shares for an object")
	}

	// Check if the object is deleted
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return rollupPermission, permissions, dbObject, NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return rollupPermission, permissions, dbObject, NewAppError(405, err, "Unallowed to set shares for deleted objects.")
		case dbObject.IsDeleted:
			return rollupPermission, permissions, dbObject, NewAppError(405, err, "Unallowed to set shares for deleted objects. Use removeObjectFromTrash to restore this object before setting shares.")
		}
	}

	//Get the json data from the request and map to an array of permission objects
	permissions, err = parseObjectShareRequest(r, ctx)
	if err != nil {
		return rollupPermission, permissions, dbObject, NewAppError(400, err, "Error parsing request")
	}

	// All good
	return rollupPermission, permissions, dbObject, nil
}

// reduceGrantsFromExistingPermissionsLeavesNone helps prevent creation of overlapping CRUDS (create, read, update, delete, share) permissions
// by comparing the permission capabilities being granted to that already being granted to the user through existing permissions and
// returns whether or not there are no capabiltiies left on the modified permission.
func reduceGrantsFromExistingPermissionsLeavesNone(existingPermissions []models.ODObjectPermission, permission *models.ODObjectPermission) bool {

	// Iterate existing permissions on object
	for _, existingPermission := range existingPermissions {
		granteeMatch := isPermissionFor(&existingPermission, permission.Grantee)
		everyoneMatch := isPermissionFor(&existingPermission, models.EveryoneGroup)
		if !existingPermission.IsDeleted && (granteeMatch || everyoneMatch) {
			// Discern which permissions this user already has,
			// removing them from the permission passed in
			if existingPermission.AllowCreate {
				permission.AllowCreate = false
			}
			if existingPermission.AllowRead {
				permission.AllowRead = false
			}
			if existingPermission.AllowUpdate {
				permission.AllowUpdate = false
			}
			if existingPermission.AllowDelete {
				permission.AllowDelete = false
			}
			if existingPermission.AllowShare {
				permission.AllowShare = false
			}
		}

		// Stop reducing if we have no permissions left after changes
		if !grantsAny(permission) {
			return true
		}
	}

	return grantsAny(permission) == false
}

func grantsAny(permission *models.ODObjectPermission) bool {
	// Determine if no permissions are remaining
	hasPermissions := false
	hasPermissions = hasPermissions || permission.AllowCreate
	hasPermissions = hasPermissions || permission.AllowRead
	hasPermissions = hasPermissions || permission.AllowUpdate
	hasPermissions = hasPermissions || permission.AllowDelete
	hasPermissions = hasPermissions || permission.AllowShare

	return hasPermissions
}

func userPermissionToShareError(missingCapability string, removing bool) *AppError {
	code := http.StatusForbidden
	action := "set"
	if removing {
		action = "remove"
	}
	err := fmt.Errorf("User does not have permission to %s share with %s", action, missingCapability)
	msg := fmt.Sprintf("Forbidden - Unauthorized to %s permissions with %s", action, missingCapability)
	return NewAppError(code, err, msg)
}
func verifyPermissionToShare(rollupPermission models.ODObjectPermission, permission models.ODObjectPermission, removing bool) *AppError {
	if !rollupPermission.AllowCreate && permission.AllowCreate {
		return userPermissionToShareError("create", removing)
	}
	if !rollupPermission.AllowRead && permission.AllowRead {
		return userPermissionToShareError("read", removing)
	}
	if !rollupPermission.AllowUpdate && permission.AllowUpdate {
		return userPermissionToShareError("update", removing)
	}
	if !rollupPermission.AllowDelete && permission.AllowDelete {
		return userPermissionToShareError("delete", removing)
	}
	if !rollupPermission.AllowShare && permission.AllowShare {
		return userPermissionToShareError("delegation", removing)
	}
	return nil
}

func isPermissionFor(permission *models.ODObjectPermission, grantee string) bool {
	return (strings.Compare(models.AACFlatten(permission.Grantee), models.AACFlatten(grantee)) == 0)
}

func removePermissionsForGrantee(obj *models.ODObject, grantee string) {
	for i := len(obj.Permissions) - 1; i >= 0; i-- {
		permission := obj.Permissions[i]
		if isPermissionFor(&permission, grantee) {
			permission.IsDeleted = true
			obj.Permissions[i] = permission
			if permission.IsCreating() {
				obj.Permissions = append(obj.Permissions[:i], obj.Permissions[i+1:]...)
			}
		}
	}
}

func hasPermissionsForGrantee(obj *models.ODObject, grantee string) bool {
	for i := len(obj.Permissions) - 1; i >= 0; i-- {
		permission := obj.Permissions[i]
		if permission.IsDeleted {
			continue
		}
		if isPermissionFor(&permission, grantee) {
			return true
		}
	}
	return false
}

func parseObjectShareRequest(r *http.Request, ctx context.Context) ([]models.ODObjectPermission, error) {
	var requestedShare protocol.ObjectShare
	var requestedPermissions []models.ODObjectPermission
	var err error

	// Parse the JSON body into the requestedShare
	err = util.FullDecode(r.Body, &requestedShare)
	if err != nil {
		return requestedPermissions, errors.New("unable to decode share from JSON body")
	}
	// Map to internal permission(s)
	requestedPermissions, err = mapping.MapObjectShareToODPermissions(&requestedShare)
	if err != nil {
		return requestedPermissions, errors.New("error mapping share to permissions")
	}

	// Return it
	return requestedPermissions, nil
}

func getObjectIDFromContext(ctx context.Context) ([]byte, error) {
	var bytesObjectID []byte
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return bytesObjectID, errors.New("could not get capture groups")
	}
	// Assign requestedPermission with the objectId being shared
	if captured["objectId"] == "" {
		return bytesObjectID, errors.New("could not extract objectid from URI")
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return bytesObjectID, errors.New("invalid objectid in URI")
	}
	return bytesObjectID, nil
}

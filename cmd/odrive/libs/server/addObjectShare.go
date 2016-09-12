package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) addObjectShare(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	session := SessionIDFromContext(ctx)

	rollupPermission, permissions, dbObject, herr := commonObjectSharePrep(ctx, h.MasterKey, r)
	if herr != nil {
		return herr
	}

	dao := DAOFromContext(ctx)
	// Only proceed if there were permissions provided
	if len(permissions) > 0 {

		var permissionsToAdd []models.ODObjectPermission

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
				}
			}
		}

		// Force Owner CRUDS
		if removingEveryone {
			ownerCRUDS := models.PermissionForUser(dbObject.OwnedBy.String, false, true, false, false, false)
			permissions = append(permissions, ownerCRUDS)
		}

		// Iterate the permissions, normalizing the share to derive grantee
		for _, permission := range permissions {

			// Verify that permission settings are allowed for user's rollupPermission
			if herr := verifyPermissionToShare(rollupPermission, permission); herr != nil {
				return herr
			}

			// Flatten grantee from the acmshare parsed in the request
			if herr := h.flattenGranteeOnPermission(ctx, &permission); herr != nil {
				return herr
			}

			// Metadata for this permission to be created
			bytesObjectID, _ := getObjectIDFromContext(ctx)
			permission.ObjectID = bytesObjectID
			permission.CreatedBy = caller.DistinguishedName
			permission.ExplicitShare = true

			// If after removing existing grants there are no more permissions, ...
			plannedPermissions := []models.ODObjectPermission{}
			plannedPermissions = append(plannedPermissions, dbObject.Permissions...)
			plannedPermissions = append(plannedPermissions, permissionsToAdd...)
			if reduceGrantsFromExistingPermissionsLeavesNone(plannedPermissions, &permission) {
				// stop processing this permission
				continue
			}

			// Now we can assign encrypt key, which will set mac based upon permissions being granted
			models.CopyEncryptKey(h.MasterKey, &rollupPermission, &permission)

			// And add it to the list of permissions that will be added
			permissionsToAdd = append(permissionsToAdd, permission)

			// For permissions granting read, merge permission.AcmShare into dbObject.RawAcm.String{share}
			if permission.AllowRead {
				herr, sourceInterface := getACMInterfacePart(&dbObject, "share")
				if herr != nil {
					return herr
				}
				interfaceToAdd, err := utils.UnmarshalStringToInterface(permission.AcmShare)
				if err != nil {
					return NewAppError(500, err, "Unable to unmarshal share from permission")
				}
				combinedInterface := CombineInterface(sourceInterface, interfaceToAdd)
				acmstring, _ := utils.MarshalInterfaceToString(combinedInterface)
				logger.Info("after combining", zap.String("new acm", acmstring))
				herr = setACMPartFromInterface(ctx, &dbObject, "share", combinedInterface)
				if herr != nil {
					return herr
				}
			}
		}

		// Update the database object now that its ACM has been altered
		dbObject.ModifiedBy = caller.DistinguishedName
		// Reflatten dbObject.RawACM
		err := h.flattenACM(logger, &dbObject)
		if err != nil {
			return NewAppError(500, err, "Error updating permissions when flattening acm")
		}

		// Now that the result is flattened, perform resultant state validation
		if !isModifiedBySameAsOwner(&dbObject) {
			// 1. Make sure owner would still have read access
			if !h.isObjectACMSharedToUser(ctx, &dbObject, dbObject.OwnedBy.String) {
				errMsg := "Forbidden - Unauthorized to set permissions that would result in owner not being able to read object"
				return NewAppError(403, errors.New(errMsg), errMsg)
			}
		} else {
			// 2. User must pass access check against altered ACM as a whole
			hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
			if err != nil {
				// TODO: Isolate different error types
				//return NewAppError(502, err, "Error communicating with authorization service")
				return NewAppError(403, err, err.Error())
			}
			if !hasAACAccess {
				return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for updated object ACM")
			}
		}

		// First update the base object that favors ACM change
		err = dao.UpdateObject(&dbObject)
		if err != nil {
			return NewAppError(500, err, "Error updating object")
		}

		// Add these permissions to the database.
		for _, permission := range permissionsToAdd {
			// Add to database
			_, err := dao.AddPermissionToObject(dbObject, &permission, false, h.MasterKey)
			if err != nil {
				return NewAppError(500, err, "Error updating permission on object - add permission")
			}
		}
	}

	// Now fetch updated object
	updatedObject, err := dao.GetObject(dbObject, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	apiResponse := mapping.MapODObjectToObject(&updatedObject).WithCallerPermission(protocolCaller(caller))
	h.EventQueue.Publish(events.Index{
		ObjectID:     apiResponse.ID,
		Timestamp:    time.Now().Format(time.RFC3339),
		Action:       "update",
		ChangeToken:  apiResponse.ChangeToken,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: false,
		SessionID:    session,
	})
	jsonResponse(w, apiResponse)

	return nil
}

func commonObjectSharePrep(ctx context.Context, masterKey string, r *http.Request) (models.ODObjectPermission, []models.ODObjectPermission, models.ODObject, *AppError) {

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
	allowedToShare, rollupPermission = isUserAllowedToShareWithPermission(ctx, masterKey, &dbObject)
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

func verifyPermissionToShare(rollupPermission models.ODObjectPermission, permission models.ODObjectPermission) *AppError {
	if !rollupPermission.AllowCreate && permission.AllowCreate {
		return NewAppError(403, fmt.Errorf("User does not have permission to set share with create"), "Forbidden - Unauthorized to set permissions with create")
	}
	if !rollupPermission.AllowRead && permission.AllowRead {
		return NewAppError(403, fmt.Errorf("User does not have permission to set share with read"), "Forbidden - Unauthorized to set permissions with read")
	}
	if !rollupPermission.AllowUpdate && permission.AllowUpdate {
		return NewAppError(403, fmt.Errorf("User does not have permission to set share with update"), "Forbidden - Unauthorized to set permissions with update")
	}
	if !rollupPermission.AllowDelete && permission.AllowDelete {
		return NewAppError(403, fmt.Errorf("User does not have permission to set share with delete"), "Forbidden - Unauthorized to set permissions with delete")
	}
	if !rollupPermission.AllowShare && permission.AllowShare {
		return NewAppError(403, fmt.Errorf("User does not have permission to set share with delegation"), "Forbidden - Unauthorized to set permissions with delegation")
	}
	return nil
}

func isModifiedBySameAsOwner(object *models.ODObject) bool {
	return (strings.Compare(object.ModifiedBy, object.OwnedBy.String) == 0)
}

func isPermissionFor(permission *models.ODObjectPermission, grantee string) bool {
	return (strings.Compare(models.AACFlatten(permission.Grantee), models.AACFlatten(grantee)) == 0)
}

func removePermissionsForGrantee(obj *models.ODObject, grantee string) {
	for i := len(obj.Permissions) - 1; i >= 0; i-- {
		permission := obj.Permissions[i]
		if isPermissionFor(&permission, grantee) {
			obj.Permissions = append(obj.Permissions[:i], obj.Permissions[i+1:]...)
		}
	}
}

func hasPermissionsForGrantee(obj *models.ODObject, grantee string) bool {
	for i := len(obj.Permissions) - 1; i >= 0; i-- {
		permission := obj.Permissions[i]
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
		return requestedPermissions, errors.New("Unable to decode share from JSON body")
	}
	// Map to internal permission(s)
	requestedPermissions, err = mapping.MapObjectShareToODPermissions(&requestedShare)
	if err != nil {
		return requestedPermissions, errors.New("Error mapping share to permissions")
	}

	// Return it
	return requestedPermissions, nil
}

func getObjectIDFromContext(ctx context.Context) ([]byte, error) {
	var bytesObjectID []byte
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return bytesObjectID, errors.New("Could not get capture groups")
	}
	// Assign requestedPermission with the objectId being shared
	if captured["objectId"] == "" {
		return bytesObjectID, errors.New("Could not extract objectid from URI")
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return bytesObjectID, errors.New("Invalid objectid in URI.")
	}
	return bytesObjectID, nil
}

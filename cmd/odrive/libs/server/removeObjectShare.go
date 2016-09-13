package server

import (
	"errors"
	"net/http"
	"strings"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/metadata/models"
	"github.com/uber-go/zap"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectShare(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	caller, _ := CallerFromContext(ctx)
	logger := LoggerFromContext(ctx)
	dao := DAOFromContext(ctx)
	session := SessionIDFromContext(ctx)
	gem, _ := GEMFromContext(ctx)

	rollupPermission, permissions, dbObject, herr := commonObjectSharePrep(ctx, h.MasterKey, r)
	if herr != nil {
		return herr
	}

	// Only proceed if there were permissions provided
	if len(permissions) == 0 {
		logger.Info("No permissions derived from share for removal.")
	} else {
		// Get values from ctx.
		var permissionsToAdd []models.ODObjectPermission
		permissionsChanged := false

		// Check if database object has a read permission for everyone
		dbHasEveryone := hasPermissionsForGrantee(&dbObject, models.EveryoneGroup)

		// Iterate the permissions, normalizing the share to derive grantee
		for _, permission := range permissions {

			// Flatten the grantee
			if herr := h.flattenGranteeOnPermission(ctx, &permission); herr != nil {
				return herr
			}
			logger.Info("Flattened.", zap.String("grantee", permission.Grantee), zap.String("acmShare", permission.AcmShare))

			// Compare to owner
			if strings.Compare(permission.Grantee, models.AACFlatten(dbObject.OwnedBy.String)) == 0 {
				errMsg := "Forbidden - Unauthorized to set permissions that would result in owner losing access"
				return NewAppError(403, errors.New(errMsg), errMsg)
			}

			// Iterate database permissions comparing grantee
			for i, dbPermission := range dbObject.Permissions {

				if dbPermission.IsDeleted {
					continue
				}

				if strings.Compare(dbPermission.Grantee, permission.Grantee) == 0 {
					permissionsChanged = true
					dbPermission.IsDeleted = true

					// If removing everyone group, then give read access to owner
					if dbHasEveryone && strings.Compare(models.EveryoneGroup, permission.AcmGrantee.GroupName.String) == 0 {
						// Add read permission for the owner
						newOwnerPermission := copyPermissionToGrantee(&dbPermission, dbObject.OwnedBy.String)
						if herr := h.flattenGranteeOnPermission(ctx, &newOwnerPermission); herr != nil {
							return herr
						}
						// Now we can assign encrypt key, which will set mac based upon permissions being granted
						models.CopyEncryptKey(h.MasterKey, &rollupPermission, &newOwnerPermission)
						permissionsToAdd = append(permissionsToAdd, newOwnerPermission)
						dbHasEveryone = false
					}
				}

				if dbPermission.IsDeleted {
					// permission changed, need to reflect in the array
					dbObject.Permissions[i] = dbPermission
				}

			} // iterate db permissions
		} // iterate permissions representing targets passed in for removal

		// If there were changes
		if permissionsChanged {

			// Force Owner CRUDS
			ownerCRUDS := models.PermissionForUser(dbObject.OwnedBy.String, true, !dbHasEveryone, true, true, true)
			plannedPermissions := []models.ODObjectPermission{}
			plannedPermissions = append(plannedPermissions, dbObject.Permissions...)
			plannedPermissions = append(plannedPermissions, permissionsToAdd...)
			if !reduceGrantsFromExistingPermissionsLeavesNone(plannedPermissions, &ownerCRUDS) {
				permissionsToAdd = append(permissionsToAdd, ownerCRUDS)
			}

			// Flatten grantees on db permission to prep for rebuilding the acm
			for i := 0; i < len(dbObject.Permissions); i++ {
				permission := dbObject.Permissions[i]
				if herr := h.flattenGranteeOnPermission(ctx, &permission); herr != nil {
					return herr
				}
				dbObject.Permissions[i] = permission
			}

			// Rebuild it
			if herr := rebuildACMShareFromObjectPermissions(ctx, &dbObject, permissionsToAdd); herr != nil {
				return herr
			}

			// Reflatten dbObject.RawACM
			if err := h.flattenACM(logger, &dbObject); err != nil {
				return NewAppError(500, err, "Error updating permissions when flattening acm")
			}

			// Assign modifier now that its ACM has been altered
			dbObject.ModifiedBy = caller.DistinguishedName

			// Verify minimal access is met
			if !isModifiedBySameAsOwner(&dbObject) {
				// Using AAC, verify that owner would still have read access
				if !h.isObjectACMSharedToUser(ctx, &dbObject, dbObject.OwnedBy.String) {
					errMsg := "Forbidden - Unauthorized to set permissions that would result in owner not being able to read object"
					return NewAppError(403, errors.New(errMsg), errMsg)
				}
			} else {
				// Using AAC, verify the caller would still have read access
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

			// Update the base object that favors ACM change
			if err := dao.UpdateObject(&dbObject); err != nil {
				return NewAppError(500, err, "Error updating object")
			}

			// Add any new permissions for owner to the database.
			for _, permission := range permissionsToAdd {
				// Add to database
				permission.CreatedBy = caller.DistinguishedName
				permission.ObjectID = dbObject.ID
				_, err := dao.AddPermissionToObject(dbObject, &permission, false, h.MasterKey)
				if err != nil {
					return NewAppError(500, err, "Error updating permission on object - add permission")
				}
			}
		}
	} // permissions provided

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
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)
	return nil
}

func rebuildACMShareFromObjectPermissions(ctx context.Context, dbObject *models.ODObject, permissionsToAdd []models.ODObjectPermission) *AppError {
	// dbObject permissions will now reflect a new state. Rebuild it
	emptyInterface, err := utils.UnmarshalStringToInterface("{}")
	if err != nil {
		return NewAppError(500, err, "Unable to unmarshal empty interface")
	}
	if herr := setACMPartFromInterface(ctx, dbObject, "share", emptyInterface); herr != nil {
		return herr
	}

	// Iterate to build new share
	for _, dbPermission := range dbObject.Permissions {
		// For permissions granting read, merge permission.AcmShare into dbObject.RawAcm.String{share}
		if dbPermission.AllowRead &&
			!dbPermission.IsDeleted &&
			!isPermissionFor(&dbPermission, models.EveryoneGroup) {
			herr, sourceInterface := getACMInterfacePart(dbObject, "share")
			if herr != nil {
				return herr
			}
			interfaceToAdd, err := utils.UnmarshalStringToInterface(dbPermission.AcmShare)
			if err != nil {
				return NewAppError(500, err, "Unable to unmarshal share from permission",
					zap.String("dbPermission.AcmShare", dbPermission.AcmShare),
					zap.String("dbPermission.Grantee", dbPermission.Grantee))
			}
			combinedInterface := CombineInterface(sourceInterface, interfaceToAdd)
			if herr = setACMPartFromInterface(ctx, dbObject, "share", combinedInterface); herr != nil {
				return herr
			}
		}
	}

	// Iterate any permissions that will be added, also combinining in
	for _, permission := range permissionsToAdd {
		// For permissions granting read, merge permission.AcmShare into dbObject.RawAcm.String{share}
		if permission.AllowRead &&
			!permission.IsDeleted &&
			!isPermissionFor(&permission, models.EveryoneGroup) {
			herr, sourceInterface := getACMInterfacePart(dbObject, "share")
			if herr != nil {
				return herr
			}
			interfaceToAdd, err := utils.UnmarshalStringToInterface(permission.AcmShare)
			if err != nil {
				return NewAppError(500, err, "Unable to unmarshal share from permission",
					zap.String("permission.AcmShare", permission.AcmShare),
					zap.String("permission.Grantee", permission.Grantee))
			}
			combinedInterface := CombineInterface(sourceInterface, interfaceToAdd)
			if herr = setACMPartFromInterface(ctx, dbObject, "share", combinedInterface); herr != nil {
				return herr
			}
		}
	}

	return nil
}

func copyPermissionToGrantee(originalPermission *models.ODObjectPermission, grantee string) models.ODObjectPermission {
	// NOTE: This will be an area that gets complicate when changeowner implemented and ability to assign ownership to groups since
	// need to maintain project name, displayname and group name

	// This call assumes grantee is a user in the form of a distinguished name (not flattened)
	return models.PermissionForUser(grantee, originalPermission.AllowCreate, originalPermission.AllowRead, originalPermission.AllowUpdate, originalPermission.AllowDelete, originalPermission.AllowShare)
}

package server

import (
	"errors"
	"net/http"
	"strings"

	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"

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

			// Rebuild it
			if herr := rebuildACMShareFromObjectPermissions(ctx, &dbObject, permissionsToAdd); herr != nil {
				return herr
			}

			// Reflatten dbObject.RawACM
			if err := h.flattenACM(ctx, &dbObject); err != nil {
				return ClassifyFlattenError(err)
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
				if err := h.isUserAllowedForObjectACM(ctx, &dbObject); err != nil {
					return ClassifyObjectACMError(err)
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

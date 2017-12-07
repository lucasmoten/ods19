package server

import (
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/deciphernow/object-drive-server/events"
	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/services/audit"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"golang.org/x/net/context"
)

func (h AppServer) removeObjectShare(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

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
		logger.Info("No permissions derived from share for removal.")
	} else {
		permissionsChanged := false

		// Check if database object has a read permission for everyone
		dbHasEveryone := hasPermissionsForGrantee(&dbObject, models.EveryoneGroup)
		removingEveryone := false

		// Owner reference
		ownerGrantee := models.NewODAcmGranteeFromResourceName(dbObject.OwnedBy.String)

		// Iterate the permissions to be removed
		for _, permission := range permissions {
			// Compare to owner, never a need to remove these
			if models.AACFlatten(permission.Grantee) == models.AACFlatten(ownerGrantee.Grantee) {
				continue
			}
			// Iterate database permissions
			for i, dbPermission := range dbObject.Permissions {
				// Nothing to do with this permission if already deleted
				if dbPermission.IsDeleted {
					continue
				}
				// Verify that permission being removed is allowed for user's rollupPermission
				if herr := verifyPermissionToShare(rollupPermission, permission, true); herr != nil {
					return herr
				}
				// If permission to be deleted represents everyone
				if models.AACFlatten(models.EveryoneGroup) == models.AACFlatten(permission.AcmGrantee.GroupName.String) {
					removingEveryone = true
				}
				// if DB permission grantee matches that on permission to be deleted ...
				if models.AACFlatten(dbPermission.Grantee) == models.AACFlatten(permission.Grantee) {
					permissionsChanged = true
					dbPermission.IsDeleted = true
					dbObject.Permissions[i] = dbPermission
				}
			} // iterate db permissions
		} // iterate permissions representing targets passed in for removal

		// If there were changes
		if permissionsChanged {
			// If removing everyone
			if dbHasEveryone && removingEveryone {
				// Need to add owner CRUDS back
				ownerCRUDS, _ := models.PermissionForOwner(dbObject.OwnedBy.String)
				plannedPermissions := []models.ODObjectPermission{}
				plannedPermissions = append(plannedPermissions, dbObject.Permissions...)
				if !reduceGrantsFromExistingPermissionsLeavesNone(plannedPermissions, &ownerCRUDS) {
					models.CopyEncryptKey(masterKey, &rollupPermission, &ownerCRUDS)
					dbObject.Permissions = append(dbObject.Permissions, ownerCRUDS)
				}
			}

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
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PERMISSION_MODIFY")
	gem.Payload.ObjectID = hex.EncodeToString(updatedObject.ID)
	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(updatedObject.ID))
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(updatedObject))
	gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)
	return nil
}

package server

import (
	"encoding/hex"
	"log"
	"strings"

	"github.com/uber-go/zap"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"golang.org/x/net/context"
)

func authHTTPErr(err error) int {
	// Type conversion to auth.Error with our error string
	switch auth.Error(err.Error()) {
	case auth.ErrACMNotSpecified,
		auth.ErrACMNotValid,
		auth.ErrACMResponseFailed:
		return 400
	case auth.ErrUserNotAuthorized:
		return 403
	case auth.ErrFailToCheckUserAccess,
		auth.ErrFailToFlattenACM,
		auth.ErrFailToInjectPermissions,
		auth.ErrFailToNormalizePermissions,
		auth.ErrFailToRebuildACMFromPermissions,
		auth.ErrFailToRetrieveSnippets:
		return 502
	default:
		log.Printf("WARNING: default 403 response due to unmapped error %v\n", err)
		return 403
	}
}

func isUserAllowedToReadWithPermission(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	return isUserAllowedTo(ctx, obj, requiredPermission, false)
}
func isUserAllowedToUpdateWithPermission(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowUpdate = true
	return isUserAllowedTo(ctx, obj, requiredPermission, true)
}
func isUserAllowedToShareWithPermission(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowShare = true
	return isUserAllowedTo(ctx, obj, requiredPermission, true)
}
func isUserAllowedToCreate(ctx context.Context, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowCreate = true
	ok, _ := isUserAllowedTo(ctx, obj, requiredPermission, true)
	return ok
}
func isUserAllowedToRead(ctx context.Context, obj *models.ODObject) bool {
	ok, _ := isUserAllowedToReadWithPermission(ctx, obj)
	return ok
}
func isUserAllowedToUpdate(ctx context.Context, obj *models.ODObject) bool {
	ok, _ := isUserAllowedToUpdateWithPermission(ctx, obj)
	return ok
}
func isUserAllowedToDelete(ctx context.Context, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowDelete = true
	ok, _ := isUserAllowedTo(ctx, obj, requiredPermission, true)
	return ok
}
func isUserAllowedToShare(ctx context.Context, obj *models.ODObject) bool {
	ok, _ := isUserAllowedToShareWithPermission(ctx, obj)
	return ok
}
func isUserAllowedTo(ctx context.Context, obj *models.ODObject, requiredPermission models.ODObjectPermission, rollup bool) (bool, models.ODObjectPermission) {
	caller, _ := CallerFromContext(ctx)
	groups, _ := GroupsFromContext(ctx)
	authorizedTo := false
	var userPermission models.ODObjectPermission
	var granteeMatch bool

	masterKey := ciphertext.FindCiphertextCacheByObject(nil).GetMasterKey()

	for _, permission := range obj.Permissions {
		// Skip if permission is deleted
		if permission.IsDeleted {
			continue
		}
		// Skip if permission does not apply to this user
		granteeMatch = false
		if models.AACFlatten(permission.Grantee) == models.AACFlatten(caller.DistinguishedName) {
			granteeMatch = true
		} else if models.AACFlatten(permission.Grantee) == models.AACFlatten(models.EveryoneGroup) {
			granteeMatch = true
		} else if strings.Compare(strings.ToLower(permission.Grantee), strings.ToLower(caller.DistinguishedName)) == 0 {
			granteeMatch = true
		} else if strings.Compare(strings.ToLower(permission.AcmGrantee.GroupName.String), strings.ToLower(models.EveryoneGroup)) == 0 {
			granteeMatch = true
		} else {
			for _, group := range groups {
				if models.AACFlatten(permission.Grantee) == models.AACFlatten(group) {
					granteeMatch = true
					break
				}
				if strings.Compare(strings.ToLower(permission.Grantee), strings.ToLower(group)) == 0 {
					granteeMatch = true
					break
				}
			}
		}
		if !granteeMatch {
			continue
		}
		// Skip if this this permission has invalid signature
		if !models.EqualsPermissionMAC(masterKey, &permission) {
			// Not valid. Log it
			LoggerFromContext(ctx).Warn("Invalid mac on permission, skipping", zap.String("objectId", hex.EncodeToString(obj.ID)), zap.String("permissionId", hex.EncodeToString(permission.ID)), zap.String("grantee", permission.Grantee))
			continue
		}

		// Check if this permission matches anything that is required
		if (requiredPermission.AllowCreate && permission.AllowCreate) ||
			(requiredPermission.AllowRead && permission.AllowRead) ||
			(requiredPermission.AllowUpdate && permission.AllowUpdate) ||
			(requiredPermission.AllowDelete && permission.AllowDelete) ||
			(requiredPermission.AllowShare && permission.AllowShare) {

			// Build up combined permission
			// TODO(cm): is this ever not going to be the case?
			if len(userPermission.Grantee) == 0 {
				// first hit, copy direct
				userPermission = permission
			} else {
				// append the grants
				userPermission.AllowCreate = userPermission.AllowCreate || permission.AllowCreate
				userPermission.AllowRead = userPermission.AllowRead || permission.AllowRead
				userPermission.AllowUpdate = userPermission.AllowUpdate || permission.AllowUpdate
				userPermission.AllowDelete = userPermission.AllowDelete || permission.AllowDelete
				userPermission.AllowShare = userPermission.AllowShare || permission.AllowShare
			}

			// Determine if all requirements met yet
			if !authorizedTo {
				authorizedTo = (!requiredPermission.AllowCreate || userPermission.AllowCreate) &&
					(!requiredPermission.AllowRead || userPermission.AllowRead) &&
					(!requiredPermission.AllowUpdate || userPermission.AllowUpdate) &&
					(!requiredPermission.AllowDelete || userPermission.AllowDelete) &&
					(!requiredPermission.AllowShare || userPermission.AllowShare)
			}

			// If authorized and dont need full rollup...
			if authorizedTo {
				// if short circuiting as soon as requirements met without a need for rollup
				if !rollup {
					// stop processing, have everything we need
					break
				} else {
					// if overall is everything
					if userPermission.AllowCreate &&
						userPermission.AllowRead &&
						userPermission.AllowUpdate &&
						userPermission.AllowDelete &&
						userPermission.AllowShare {
						// stop processing, no need to combine more permissions
						break
					}
				}
			}
		} // if permission matches on something required

	} // Iterate permissions

	// Recalculate the MAC for the derived permission
	userPermission.PermissionMAC = models.CalculatePermissionMAC(masterKey, &userPermission)

	// Return the overall (either combined from one or more granteeMatch, or empty from no match)
	return authorizedTo, userPermission
}

func copyPermissionToGrantee(originalPermission *models.ODObjectPermission, grantee models.ODAcmGrantee) models.ODObjectPermission {
	dn := grantee.UserDistinguishedName.String
	pn := grantee.ProjectName.String
	pdn := grantee.ProjectDisplayName.String
	gn := grantee.GroupName.String
	if len(grantee.UserDistinguishedName.String) > 0 {
		return models.PermissionForUser(dn, originalPermission.AllowCreate, originalPermission.AllowRead, originalPermission.AllowUpdate, originalPermission.AllowDelete, originalPermission.AllowShare)
	}
	return models.PermissionForGroup(pn, pdn, gn, originalPermission.AllowCreate, originalPermission.AllowRead, originalPermission.AllowUpdate, originalPermission.AllowDelete, originalPermission.AllowShare)
}

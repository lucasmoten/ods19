package server

import (
	"encoding/hex"
	"strings"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/metadata/models"
	"golang.org/x/net/context"
)

// ClassifyObjectACMError is the default pattern for classifying errors
func ClassifyObjectACMError(err error) *AppError {
	if IsDeniedAccess(err) {
		return NewAppError(403, err, err.Error())
	}
	if IsUserAllowedForObjectInternalError(err) {
		return NewAppError(502, err, err.Error())
	}
	return NewAppError(400, err, err.Error())
}

// IsDeniedAccess is for access denials that have nothing to do with internal or input errors
func IsDeniedAccess(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "auth: user not authorized")
}

// IsUserAllowedForObjectInternalError is for failures to even make the call
func IsUserAllowedForObjectInternalError(err error) bool {
	if err == nil {
		return false
	}
	if strings.HasPrefix(err.Error(), "auth: unable") {
		return true
	}
	if strings.HasPrefix(err.Error(), "auth: service") {
		return true
	}
	return false
}

// ClassifyFlattenError is the default pattern for classifying errors
func ClassifyFlattenError(err error) *AppError {
	if IsFlattenInternalError(err) {
		return NewAppError(500, err, err.Error())
	}
	return NewAppError(400, err, err.Error())
}

// IsFlattenInternalError indicates an internal error during flattening, not bad user input
func IsFlattenInternalError(err error) bool {
	if err == nil {
		return false
	}
	if strings.HasPrefix(err.Error(), "auth: unable") {
		return true
	}
	if strings.HasPrefix(err.Error(), "auth: service") {
		return true
	}
	return false
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

	dp := ciphertext.FindCiphertextCacheByObject(obj)
	masterKey := dp.GetMasterKey()

	for _, permission := range obj.Permissions {
		//LoggerFromContext(ctx).Info("Examining permissions ", zap.Object("permission", permission))
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
			// LoggerFromContext(ctx).Info("Grantee is not a match", zap.String("grantee", permission.Grantee))
			continue
		}
		// LoggerFromContext(ctx).Info("Grantee matches", zap.String("grantee", permission.Grantee))
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

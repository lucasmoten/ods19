package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/uber-go/zap"

	globalconfig "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/utils"
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
	if strings.HasPrefix(err.Error(), "Access Denied") {
		return true
	}
	return false
}

// IsUserAllowedForObjectInternalError is for failures to even make the call
func IsUserAllowedForObjectInternalError(err error) bool {
	if err == nil {
		return false
	}
	if strings.HasPrefix(err.Error(), "Error calling AAC.CheckAccess") {
		return true
	}
	return false
}

func (h AppServer) isUserAllowedForObjectACM(ctx context.Context, object *models.ODObject) error {
	var err error

	caller, ok := CallerFromContext(ctx)
	if !ok {
		return errors.New("Could not determine user")
	}

	// Validate object
	if object == nil {
		return errors.New("Object passed in is not initialized")
	}
	if !object.RawAcm.Valid {
		return errors.New("Object passed in does not have an ACM set")
	}

	// Gather inputs
	tokenType := "pki_dias"
	dn := caller.DistinguishedName
	acm := object.RawAcm.String

	logger := LoggerFromContext(ctx).With(zap.String("dn", dn), zap.String("acm", acm))

	// Verify we have a reference to AAC
	if h.AAC == nil {
		return errors.New("AAC field is nil")
	}

	// Performance instrumentation
	var beganAt = performance.BeganJob(int64(0))
	if h.Tracker != nil {
		beganAt = h.Tracker.BeginTime(performance.AACCounterCheckAccess)
	}

	// Call AAC
	aacResponse, err := h.AAC.CheckAccess(dn, tokenType, acm)

	// End performance tracking for the AAC call
	if h.Tracker != nil {
		h.Tracker.EndTime(performance.AACCounterCheckAccess, beganAt, performance.SizeJob(1))
	}

	// Check if there was an error calling the service
	if err != nil {
		logger.Error("Error calling AAC.CheckAccess", zap.String("err", err.Error()))
		return errors.New("Error calling AAC.CheckAccess")
	}
	//Note: err == nil after this point, making err.Error() a guaranteed crash

	// This is some kind of internal error if it happens
	if aacResponse == nil {
		logger.Error("Error calling AAC.CheckAccess")
		return errors.New("Error calling AAC.CheckAccess")
	}

	// Process Response
	// Log the messages
	for _, message := range aacResponse.Messages {
		logger.Error("Message in AAC Response", zap.String("acm message", message))
	}
	msgsString := strings.Join(aacResponse.Messages, "/")
	logger = logger.With(zap.String("messages", msgsString))

	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !aacResponse.Success {
		logger.Error("aacResponse.Success == false")
		return fmt.Errorf("Response from AAC.CheckAccess failed: %s", msgsString)
	}
	// AAC Response returned without error, was successful
	if !aacResponse.HasAccess {
		logger.Error("aacResponse.HasAccess == false")
		return fmt.Errorf("Access Denied for ACM: %s", msgsString)
	}
	return nil
}

// isUserAllowedForACMSTring wraps isUserAllowedForObjectACM to help with the cases where we simply need to
// set up a new models.ODObject with a RawAcm and send a request to AAC.
func (h AppServer) isUserAllowedForACMString(ctx context.Context, acm string) error {
	// Ensure user is allowed this acm
	updateObjectRequest := models.ODObject{}
	updateObjectRequest.RawAcm.String = acm
	updateObjectRequest.RawAcm.Valid = true
	return h.isUserAllowedForObjectACM(ctx, &updateObjectRequest)
}

func (h AppServer) flattenACMAndCheckAccess(ctx context.Context, object *models.ODObject) (bool, error) {
	logger := LoggerFromContext(ctx)

	var err error

	// Validate object
	if object == nil {
		return false, errors.New("Object passed in is not initialized")
	}
	if !object.RawAcm.Valid {
		return false, errors.New("Object passed in does not have an ACM set")
	}

	// Gather inputs
	acm := object.RawAcm.String
	if len(acm) == 0 {
		return false, errors.New("Ther was no ACM value on the object")
	}

	// Verify we have a reference to AAC
	if h.AAC == nil {
		return false, errors.New("AAC field is nil")
	}

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return false, errors.New("Could not determine user")
	}

	// Prep to call AAC
	userToken := caller.DistinguishedName
	tokenType := "pki_dias"
	herr, acmPartShare := getACMInterfacePart(object, "share")
	if herr != nil {
		return false, herr.Error
	}
	share, err := utils.MarshalInterfaceToString(acmPartShare)
	if err != nil {
		return false, fmt.Errorf("Unable to marshal share from acm to string format: %v", err)
	}

	acmInfo := aac.AcmInfo{Path: "X", Acm: acm, IncludeInRollup: false}
	acmInfoList := []*aac.AcmInfo{&acmInfo}
	calculateRollup := false
	shareType := "other"

	// Performance instrumentation
	var beganAt = performance.BeganJob(int64(0))
	if h.Tracker != nil {
		beganAt = h.Tracker.BeginTime(performance.AACCounterCheckAccessAndPopulate)
	}

	// TODO: This is the call..
	aacResponse, err := h.AAC.CheckAccessAndPopulate(userToken, tokenType, acmInfoList, calculateRollup, shareType, share)

	// End performance tracking for the AAC call
	if h.Tracker != nil {
		h.Tracker.EndTime(
			performance.AACCounterPopulateAndValidateResponse,
			beganAt,
			performance.SizeJob(1),
		)
	}

	//ACM and dn are *always* logged!!
	logger = logger.With(zap.String("acm", acm), zap.String("dn", userToken))

	// Log the messages
	var msgsString string
	if aacResponse != nil {
		for _, message := range aacResponse.Messages {
			logger.Error("Message in AAC Response", zap.String("acm message", message))
		}
		msgsString = strings.Join(aacResponse.Messages, "/")
	}

	// Check if there was an error calling the service
	if err != nil {
		logger.Error("CheckAccessAndPopulate error")
		return false, fmt.Errorf("Error calling AAC.CheckAccessAndPopulate: %s, %s", err.Error(), msgsString)
	}

	// Process Response
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if aacResponse == nil || !aacResponse.Success {
		logger.Error("aacResponse.Success == false")
		return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed: %s", msgsString)
	}

	// Iterate response list
	if len(aacResponse.AcmResponseList) > 0 {
		for acmResponseIdx, acmResponse := range aacResponse.AcmResponseList {
			loggerIdx := logger.With(zap.Int("acmResponseIdx", acmResponseIdx))
			// Messages
			for acmMessageIdx, acmResponseMsg := range acmResponse.Messages {
				loggerIdx.Warn("acm response", zap.Int("acmMessageIdx", acmMessageIdx), zap.String("acmResponseMsg", acmResponseMsg))
			}
			// Check if successful
			if !acmResponse.Success {
				loggerIdx.Error("acmResponse.Success == false")
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed for #%d: %s, %s", acmResponseIdx, acm, msgsString)
			}
			// Check if valid
			if !acmResponse.AcmValid {
				loggerIdx.Error("acmResponse.AcmValid == false")
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates acm not valid for #%d: %s, %s", acmResponseIdx, acm, msgsString)
			}
			// Check if user has access
			if !acmResponse.HasAccess {
				loggerIdx.Error("acmResponse.HasAccess == false")
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates user does not have access to #%d: %s, %s", acmResponseIdx, acm, msgsString)
			}
			// Capture revised acm string (last in wins. but should be only 1)
			object.RawAcm.String = acmResponse.AcmInfo.Acm
		}
	} else {
		// no acm response
		logger.Warn("acm checked")
		return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate did not result in an ACM being returned: %s", msgsString)
	}

	if aacResponse.RollupAcmResponse != nil {
		acmResponse := aacResponse.RollupAcmResponse
		// Messages
		for acmMessageIdx, acmResponseMsg := range acmResponse.Messages {
			logger.Warn("aac rollup RootupAcmResponse message", zap.Int("acmMessageIdx", acmMessageIdx), zap.String("acmResponseMsg", acmResponseMsg))
		}
		// Check if successful
		if !acmResponse.Success {
			logger.Error("aac rollup acmResponse == false")
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed for RollupAcmResponse: %s, %s", acm, msgsString)
		}
		// Check if valid
		if !acmResponse.AcmValid {
			logger.Error("aac rollup acmResponse.AcmValid == false")
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates acm not valid for RollupAcmResponse: %s, %s", acm, msgsString)
		}
		// Check if user has access
		if !acmResponse.HasAccess {
			logger.Error("aac rollup acmResponse.HasAccess == false")
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates user does not have access to RollupAcmResponse: %s, %s", acm, msgsString)
		}
		// Capture revised acm string (last in wins. but should be only 1)
		object.RawAcm.String = acmResponse.AcmInfo.Acm
	}

	// Done, implicitly true
	return true, nil
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
	if strings.HasPrefix(err.Error(), "Error calling AAC.PopulateAndValidateAcm") {
		return true
	}
	if strings.HasPrefix(err.Error(), "Response from AAC.PopulateAndValidateAcm failed") {
		return true
	}
	return false
}

func (h AppServer) flattenACM(ctx context.Context, object *models.ODObject) error {

	var err error
	logger := LoggerFromContext(ctx)

	// Validate object
	if object == nil {
		return errors.New("Object passed in is not initialized")
	}
	if !object.RawAcm.Valid {
		return errors.New("Object passed in does not have an ACM set")
	}

	// Gather inputs
	acm := object.RawAcm.String
	logger = logger.With(zap.String("acm", acm))

	// Verify we have a reference to AAC
	if h.AAC == nil {
		return errors.New("AAC field is nil")
	}

	// Performance instrumentation
	var beganAt = performance.BeganJob(int64(0))
	if h.Tracker != nil {
		beganAt = h.Tracker.BeginTime(performance.AACCounterPopulateAndValidateResponse)
	}

	// Call AAC
	acmResponse, err := h.AAC.PopulateAndValidateAcm(acm)

	// End performance tracking for the AAC call
	if h.Tracker != nil {
		h.Tracker.EndTime(
			performance.AACCounterPopulateAndValidateResponse,
			beganAt,
			performance.SizeJob(1),
		)
	}

	// Check if there was an error calling the service
	if err != nil {
		logger.Error("Error calling AAC.PopulateAndValidateAcm", zap.String("err", err.Error()))
		return fmt.Errorf("Error calling AAC.PopulateAndValidateAcm: %s", err.Error())
	}

	// Prevent potential nil ptr in next block
	if acmResponse == nil {
		logger.Error("Error calling AAC.PopulateAndValidateAcm")
		return fmt.Errorf("Error calling AAC.PopulateAndValidateAcm")
	}

	// Process Response
	// Log the messages
	var msgsString string
	for _, message := range acmResponse.Messages {
		logger.Error("Message in AAC Response", zap.String("aac message", message))
	}
	msgsString = strings.Join(acmResponse.Messages, "/")

	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !acmResponse.Success {
		logger.Error("acmResponse.Success == false")
		return fmt.Errorf("Response from AAC.PopulateAndValidateAcm failed: %s", msgsString)
	}
	// Check if the acm was valid
	if !acmResponse.AcmValid {
		logger.Error("acmResponse.Valid == false")
		return fmt.Errorf("ACM in call to PopulateAndValidateAcm was not valid: %s", msgsString)
	}

	// Get revised acm string
	object.RawAcm.String = acmResponse.AcmInfo.Acm
	// Done
	return nil
}

func isUserAllowedToReadWithPermission(ctx context.Context, masterKey string, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	return isUserAllowedTo(ctx, masterKey, obj, requiredPermission, false)
}
func isUserAllowedToUpdateWithPermission(ctx context.Context, masterKey string, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowUpdate = true
	return isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
}
func isUserAllowedToShareWithPermission(ctx context.Context, masterKey string, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowShare = true
	return isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
}
func isUserAllowedToCreate(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowCreate = true
	ok, _ := isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
	return ok
}
func isUserAllowedToRead(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	ok, _ := isUserAllowedToReadWithPermission(ctx, masterKey, obj)
	return ok
}
func isUserAllowedToUpdate(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	ok, _ := isUserAllowedToUpdateWithPermission(ctx, masterKey, obj)
	return ok
}
func isUserAllowedToDelete(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{}
	requiredPermission.AllowRead = true
	requiredPermission.AllowDelete = true
	ok, _ := isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
	return ok
}
func isUserAllowedToShare(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	ok, _ := isUserAllowedToShareWithPermission(ctx, masterKey, obj)
	return ok
}
func isUserAllowedTo(ctx context.Context, masterKey string, obj *models.ODObject, requiredPermission models.ODObjectPermission, rollup bool) (bool, models.ODObjectPermission) {
	caller, _ := CallerFromContext(ctx)
	groups, _ := GroupsFromContext(ctx)
	authorizedTo := false
	var userPermission models.ODObjectPermission
	var granteeMatch bool
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

func (h AppServer) newContextWithGroupsAndSnippetsFromUser(distinguishedName string) context.Context {
	userCtx := context.Background()
	userCaller := Caller{DistinguishedName: distinguishedName}
	userCtx = ContextWithCaller(userCtx, userCaller)
	userCtx = context.WithValue(userCtx, Logger, globalconfig.RootLogger)
	userGroups, userSnippets, err := h.GetUserGroupsAndSnippets(userCtx)
	if err != nil {
		userGroups = []string{models.AACFlatten(distinguishedName)}
	}
	userCtx = ContextWithGroups(userCtx, userGroups)
	userCtx = ContextWithSnippets(userCtx, userSnippets)
	return userCtx
}

func (h AppServer) isObjectACMSharedToUser(ctx context.Context, obj *models.ODObject, user string) bool {

	// Look at the flattened share of the acm

	var acmGrants []string
	herr, fShareInterface := getACMInterfacePart(obj, "f_share")
	if herr != nil {
		LoggerFromContext(ctx).Warn("Error retrieving acm interface part f_share")
		return false
	}
	acmGrants = getStringArrayFromInterface(fShareInterface)

	//  If no values, its shared to everyone
	if len(acmGrants) == 0 {
		return true
	}

	// Since there are values, we need to check user groups

	// Default to groups from context
	userGroups, _ := GroupsFromContext(ctx)
	// If caller is not the same as user we are checking ..
	caller, _ := CallerFromContext(ctx)
	if strings.Compare(caller.DistinguishedName, user) != 0 {
		// Populate for user and get their groups
		userCtx := h.newContextWithGroupsAndSnippetsFromUser(user)
		userGroups, _ = GroupsFromContext(userCtx)
	}

	// Iterate user's groups
	for _, userGroup := range userGroups {
		// Iterate acm grants
		for _, acmGrant := range acmGrants {
			// Do they match?
			if strings.Compare(userGroup, acmGrant) == 0 {
				return true
			}
		}
	}

	// None of the user groups matched the acm. They wont have read access.
	return false
}

// rebuildACMShareFromObjectPermissions will clear the ACM share and reconstruct it from the current permissions on the object, and optional
// ones passed in that grant read access and are not marked as deleted.
func rebuildACMShareFromObjectPermissions(ctx context.Context, dbObject *models.ODObject, permissionsToAdd []models.ODObjectPermission) *AppError {
	var emptyInterface interface{}
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
			combinedInterface := CombineInterface(ctx, sourceInterface, interfaceToAdd)
			if herr = setACMPartFromInterface(ctx, dbObject, "share", combinedInterface); herr != nil {
				return herr
			}
		} else {
			LoggerFromContext(ctx).Debug("DB Permission not combined into share as it does not allow read or is deleted or is for everyone", zap.Bool("allowRead", dbPermission.AllowRead), zap.Bool("isDeleted", dbPermission.IsDeleted), zap.Bool("everyone", !isPermissionFor(&dbPermission, models.EveryoneGroup)))
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
			combinedInterface := CombineInterface(ctx, sourceInterface, interfaceToAdd)
			if herr = setACMPartFromInterface(ctx, dbObject, "share", combinedInterface); herr != nil {
				return herr
			}
		} else {
			LoggerFromContext(ctx).Debug("New Permission not combined into share as it does not allow read or is deleted or is for everyone", zap.Bool("allowRead", permission.AllowRead), zap.Bool("isDeleted", permission.IsDeleted), zap.Bool("everyone", !isPermissionFor(&permission, models.EveryoneGroup)))
		}

	}

	return nil
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

package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
)

func (h AppServer) isUserAllowedForObjectACM(ctx context.Context, object *models.ODObject) (bool, error) {
	logger := LoggerFromContext(ctx)
	// TODO: Change this to user    h.AAC.CheckAccessAndPopulate

	var err error
	// In standalone, we are ignoring AAC
	if config.StandaloneMode {
		// But warn in STDOUT to draw attention
		logger.Warn("WARNING: STANDALONE mode is active. User permission to access objects are not being checked against AAC.")
		// Return permission granted and no errors
		return true, nil
	}

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return false, errors.New("Could not determine user")
	}

	// Validate object
	if object == nil {
		return false, errors.New("Object passed in is not initialized")
	}
	if !object.RawAcm.Valid {
		return false, errors.New("Object passed in does not have an ACM set")
	}

	// Gather inputs
	tokenType := "pki_dias"
	dn := caller.DistinguishedName
	acm := object.RawAcm.String

	// Verify we have a reference to AAC
	if h.AAC == nil {
		return false, errors.New("AAC field is nil")
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
		h.Tracker.EndTime(
			performance.AACCounterCheckAccess,
			beganAt,
			performance.SizeJob(1),
		)
	}

	// Check if there was an error calling the service
	if err != nil {
		logger.Error("Error calling AAC.CheckAccess", zap.String("err", err.Error()), zap.String("acm", acm), zap.String("dn", dn))
		return false, errors.New("Error calling AAC.CheckAccess")
	}

	// Process Response
	// Log the messages
	for _, message := range aacResponse.Messages {
		logger.Error("Message in AAC Response", zap.String("acm message", message))
	}
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !aacResponse.Success {
		logger.Error("aacResponse.Success == false", zap.String("acm", acm), zap.String("dn", dn))
		return false, fmt.Errorf("Response from AAC.CheckAccess failed: %s", err.Error())
	}
	// AAC Response returned without error, was successful
	if !aacResponse.HasAccess {
		logger.Error("aacResponse.HasAccess == false", zap.String("acm", acm), zap.String("dn", dn))
	}
	return aacResponse.HasAccess, nil
}

func (h AppServer) flattenACMAndCheckAccess(ctx context.Context, object *models.ODObject) (bool, error) {
	logger := LoggerFromContext(ctx)

	var err error
	// In standalone, we are ignoring AAC
	if config.StandaloneMode {
		// But warn in STDOUT to draw attention
		logger.Warn("WARNING: STANDALONE mode is active.  ACM will not be flattened.")
		// Return permission granted and no errors
		return true, nil
	}

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

	// // Remove the share part from acm to be checked and populated if it has content
	// if len(share) > 0 {
	// 	tempObject := models.ODObject{}
	// 	tempObject.RawAcm.String = object.RawAcm.String
	// 	tempObject.RawAcm.Valid = true
	// 	setACMPartFromInterface(ctx, &tempObject, "share", nil)
	// 	acm = tempObject.RawAcm.String
	// 	log.Printf("Changing ACM being checked to: %s", acm)
	// }

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

	// Log the messages
	if aacResponse != nil {
		for _, message := range aacResponse.Messages {
			logger.Error("Message in AAC Response", zap.String("acm message", message))
		}
	}

	// Check if there was an error calling the service
	if err != nil {
		log.Printf("ACM checked: %s\n", acm)
		return false, fmt.Errorf("Error calling AAC.CheckAccessAndPopulate: %s", err.Error())
	}

	//ACM and dn are *always* logged!!
	logger = logger.With(zap.String("acm", acm), zap.String("dn", userToken))

	// Process Response
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !aacResponse.Success {
		logger.Error("aacResponse.Success == false")
		return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed: %s", err.Error())
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
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed for #%d: %s", acmResponseIdx, acm)
			}
			// Check if valid
			if !acmResponse.AcmValid {
				loggerIdx.Error("acmResponse.AcmValid == false")
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates acm not valid for #%d: %s", acmResponseIdx, acm)
			}
			// Check if user has access
			if !acmResponse.HasAccess {
				loggerIdx.Error("acmResponse.HasAccess == false")
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates user does not have access to #%d: %s", acmResponseIdx, acm)
			}
			// Capture revised acm string (last in wins. but should be only 1)
			object.RawAcm.String = acmResponse.AcmInfo.Acm
		}
	} else {
		// no acm response
		logger.Warn("acm checked")
		return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate did not result in an ACM being returned")
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
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed for RollupAcmResponse: %s", acm)
		}
		// Check if valid
		if !acmResponse.AcmValid {
			logger.Error("aac rollup acmResponse.AcmValid == false")
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates acm not valid for RollupAcmResponse: %s", acm)
		}
		// Check if user has access
		if !acmResponse.HasAccess {
			logger.Error("aac rollup acmResponse.HasAccess == false")
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates user does not have access to RollupAcmResponse: %s", acm)
		}
		// Capture revised acm string (last in wins. but should be only 1)
		object.RawAcm.String = acmResponse.AcmInfo.Acm
	}

	// Done, implicitly true
	return true, nil
}

func (h AppServer) flattenACM(logger zap.Logger, object *models.ODObject) error {

	var err error
	// In standalone, we are ignoring AAC
	if config.StandaloneMode {
		// But warn in STDOUT to draw attention
		logger.Warn("WARNING: STANDALONE mode is active.  ACM will not be flattened.")
		// Return permission granted and no errors
		return nil
	}

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
		return errors.New("Error calling AAC.PopulateAndValidateAcm")
	}

	// Process Response
	// Log the messages
	for _, message := range acmResponse.Messages {
		logger.Error("Message in AAC Response", zap.String("aac message", message))
	}
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !acmResponse.Success {
		logger.Error("acmResponse.Success == false")
		return errors.New("Response from AAC.PopulateAndValidateAcm failed")
	}
	// Check if the acm was valid
	if !acmResponse.AcmValid {
		logger.Error("acmResponse.Valid == false")
		return errors.New("ACM in call to PopulateAndValidateAcm was not valid")
	}

	// Get revised acm string
	object.RawAcm.String = acmResponse.AcmInfo.Acm

	// Done
	return nil
}

func (h AppServer) validateAndFlattenShare(ctx context.Context, permission *models.ODObjectPermission, object *models.ODObject) *AppError {
	logger := LoggerFromContext(ctx)

	// Reference to original acm
	//originalAcm := object.RawAcm.String

	// Remove existing f_share from the ACM
	herr := removeACMPart(ctx, object, "f_share")
	if herr != nil {
		return herr
	}

	// Convert the AcmShare on the permission to an interface and assign to ACM
	shareInterface, err := utils.UnmarshalStringToInterface(permission.AcmShare)
	if err != nil {
		logger.Error("unable to marshal share from permission", zap.String("permission acm share", permission.AcmShare), zap.String("err", err.Error()))
		return NewAppError(500, err, "Unable to unmarshal share from permission")
	}
	herr = setACMPartFromInterface(ctx, object, "share", shareInterface)
	if herr != nil {
		return herr
	}

	// Flatten
	h.flattenACM(logger, object)

	// Get the share part back out since its been flattened as AAC alters it
	herr, newShareInterface := getACMInterfacePart(object, "share")
	if herr != nil {
		return herr
	}
	// And then assign back on the permission
	marshalledShare, err := utils.MarshalInterfaceToString(newShareInterface)
	if err != nil {
		return NewAppError(500, err, "Unable to marshal share from flattened interface")
	}
	permission.AcmShare = marshalledShare

	return nil
}

func isUserAllowedToCreate(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{AllowCreate: true, AllowRead: true}
	ok, _ := isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
	return ok
}
func isUserAllowedToRead(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{AllowRead: true}
	ok, _ := isUserAllowedTo(ctx, masterKey, obj, requiredPermission, false)
	return ok
}
func isUserAllowedToReadWithPermission(ctx context.Context, masterKey string, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowRead: true}
	return isUserAllowedTo(ctx, masterKey, obj, requiredPermission, false)
}
func isUserAllowedToUpdate(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowUpdate: true}
	ok, _ := isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
	return ok
}
func isUserAllowedToUpdateWithPermission(ctx context.Context, masterKey string, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowUpdate: true}
	return isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
}
func isUserAllowedToDelete(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowDelete: true}
	ok, _ := isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
	return ok
}
func isUserAllowedToShareWithPermission(ctx context.Context, masterKey string, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowShare: true}
	return isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
}
func isUserAllowedToShare(ctx context.Context, masterKey string, obj *models.ODObject) bool {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowShare: true}
	ok, _ := isUserAllowedTo(ctx, masterKey, obj, requiredPermission, true)
	return ok
}

func isUserAllowedTo(ctx context.Context, masterKey string, obj *models.ODObject, requiredPermission models.ODObjectPermission, rollup bool) (bool, models.ODObjectPermission) {
	caller, _ := CallerFromContext(ctx)
	groups, _ := GroupsFromContext(ctx)
	authorizedTo := false
	var userPermission models.ODObjectPermission
	var granteeMatch bool
	for _, permission := range obj.Permissions {
		// Skip if permission is deleted
		if permission.IsDeleted {
			continue
		}
		// Skip if permission does not apply to this user
		granteeMatch = false
		if strings.Compare(strings.ToLower(permission.Grantee), strings.ToLower(caller.DistinguishedName)) == 0 {
			granteeMatch = true
		} else if strings.Compare(strings.ToLower(permission.Grantee), strings.ToLower(models.EveryoneGroup)) == 0 {
			granteeMatch = true
		} else {
			for _, group := range groups {
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
			LoggerFromContext(ctx).Warn("invalid mac on permission, skipping", zap.String("objectId", hex.EncodeToString(obj.ID)), zap.String("permissionId", hex.EncodeToString(permission.ID)), zap.String("grantee", permission.Grantee))
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

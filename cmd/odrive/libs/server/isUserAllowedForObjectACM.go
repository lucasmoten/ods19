package server

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
)

func (h AppServer) isUserAllowedForObjectACM(ctx context.Context, object *models.ODObject) (bool, error) {

	// TODO: Change this to user    h.AAC.CheckAccessAndPopulate

	var err error
	// In standalone, we are ignoring AAC
	if config.StandaloneMode {
		// But warn in STDOUT to draw attention
		log.Printf("WARNING: STANDALONE mode is active. User permission to access objects are not being checked against AAC.")
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
		log.Printf("Error calling AAC.CheckAccess: %s. ACM checked: %s\n", err.Error(), acm)
		return false, errors.New("Error calling AAC.CheckAccess")
	}

	// Process Response
	// Log the messages
	for _, message := range aacResponse.Messages {
		log.Printf("Message in AAC Response: %s\n", message)
	}
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !aacResponse.Success {
		log.Printf("aacResponse.Success = false. ACM checked: %s\n", acm)
		return false, fmt.Errorf("Response from AAC.CheckAccess failed: %s", err.Error())
	}
	// AAC Response returned without error, was successful
	if !aacResponse.HasAccess {
		log.Printf("aacResponse.HasAccess = false. ACM checked: %s\n", acm)
	}
	return aacResponse.HasAccess, nil
}

func (h AppServer) flattenACMAndCheckAccess(ctx context.Context, object *models.ODObject) (bool, error) {
	var err error
	// In standalone, we are ignoring AAC
	if config.StandaloneMode {
		// But warn in STDOUT to draw attention
		log.Printf("WARNING: STANDALONE mode is active.  ACM will not be flattened.")
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
			log.Printf("Message in AAC Response: %s\n", message)
		}
	}

	// Check if there was an error calling the service
	if err != nil {
		log.Printf("ACM checked: %s\n", acm)
		return false, fmt.Errorf("Error calling AAC.CheckAccessAndPopulate: %s", err.Error())
	}

	// Process Response
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !aacResponse.Success {
		log.Printf("ACM checked: %s\n", acm)
		return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed: %s", err.Error())
	}
	// Iterate response list
	if len(aacResponse.AcmResponseList) > 0 {
		for acmResponseIdx, acmResponse := range aacResponse.AcmResponseList {
			// Messages
			for acmMessageIdx, acmResponseMsg := range acmResponse.Messages {
				log.Printf("Message in AAC Response %d, Message #%d: %s\n", acmResponseIdx, acmMessageIdx, acmResponseMsg)
			}
			// Check if successful
			if !acmResponse.Success {
				log.Printf("ACM Response failed in %d. ACM checked: %s\n", acmResponseIdx, acm)
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed for #%d: %s", acmResponseIdx, acm)
			}
			// Check if valid
			if !acmResponse.AcmValid {
				log.Printf("ACM was not valid in %d. ACM checked: %s\n", acmResponseIdx, acm)
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates acm not valid for #%d: %s", acmResponseIdx, acm)
			}
			// Check if user has access
			if !acmResponse.HasAccess {
				log.Printf("User does not have access to acm in %d. ACM checked: %s\n", acmResponseIdx, acm)
				return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates user does not have access to #%d: %s", acmResponseIdx, acm)
			}
			// Capture revised acm string (last in wins. but should be only 1)
			object.RawAcm.String = acmResponse.AcmInfo.Acm
		}
	} else {
		// no acm response
		log.Printf("ACM checked: %s\n", acm)
		return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate did not result in an ACM being returned")
	}

	if aacResponse.RollupAcmResponse != nil {
		acmResponse := aacResponse.RollupAcmResponse
		// Messages
		for acmMessageIdx, acmResponseMsg := range acmResponse.Messages {
			log.Printf("Message in AAC RollupAcmResponse, Message #%d: %s\n", acmMessageIdx, acmResponseMsg)
		}
		// Check if successful
		if !acmResponse.Success {
			log.Printf("ACM Response failed in RollupAcmResponse. ACM checked: %s\n", acm)
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate failed for RollupAcmResponse: %s", acm)
		}
		// Check if valid
		if !acmResponse.AcmValid {
			log.Printf("ACM was not valid in RollupAcmResponse. ACM checked: %s\n", acm)
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates acm not valid for RollupAcmResponse: %s", acm)
		}
		// Check if user has access
		if !acmResponse.HasAccess {
			log.Printf("User does not have access to acm in RollupAcmResponse. ACM checked: %s\n", acm)
			return false, fmt.Errorf("Response from AAC.CheckAccessAndPopulate indicates user does not have access to RollupAcmResponse: %s", acm)
		}
		// Capture revised acm string (last in wins. but should be only 1)
		object.RawAcm.String = acmResponse.AcmInfo.Acm
	}

	// Done, implicitly true
	return true, nil
}

func (h AppServer) flattenACM(object *models.ODObject) error {

	var err error
	// In standalone, we are ignoring AAC
	if config.StandaloneMode {
		// But warn in STDOUT to draw attention
		log.Printf("WARNING: STANDALONE mode is active.  ACM will not be flattened.")
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
		log.Printf("Error calling AAC.PopulateAndValidateAcm: %s", err.Error())
		return errors.New("Error calling AAC.PopulateAndValidateAcm")
	}

	// Process Response
	// Log the messages
	for _, message := range acmResponse.Messages {
		log.Printf("Message in AAC Response: %s\n", message)
	}
	// Check if response was successful
	// -- This is assumed to be an upstream error, not a user authorization error
	if !acmResponse.Success {
		return errors.New("Response from AAC.PopulateAndValidateAcm failed")
	}
	// Check if the acm was valid
	if !acmResponse.AcmValid {
		return errors.New("ACM in call to PopulateAndValidateAcm was not valid")
	}

	// Get revised acm string
	object.RawAcm.String = acmResponse.AcmInfo.Acm

	// Done
	return nil
}

func (h AppServer) validateAndFlattenShare(ctx context.Context, permission *models.ODObjectPermission, object *models.ODObject) *AppError {
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
		log.Printf("Unable to marshal share from permission %s: %v", permission.AcmShare, err)
		return NewAppError(500, err, "Unable to unmarshal share from permission")
	}
	herr = setACMPartFromInterface(ctx, object, "share", shareInterface)
	if herr != nil {
		return herr
	}

	// Flatten
	h.flattenACM(object)

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

func isUserAllowedToCreate(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowCreate: true, AllowRead: true}
	return isUserAllowedTo(ctx, obj, requiredPermission, false)
}
func isUserAllowedToRead(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowRead: true}
	return isUserAllowedTo(ctx, obj, requiredPermission, false)
}
func isUserAllowedToUpdate(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowUpdate: true}
	return isUserAllowedTo(ctx, obj, requiredPermission, false)
}
func isUserAllowedToDelete(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowDelete: true}
	return isUserAllowedTo(ctx, obj, requiredPermission, false)
}
func isUserAllowedToShare(ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	requiredPermission := models.ODObjectPermission{AllowRead: true, AllowShare: true}
	return isUserAllowedTo(ctx, obj, requiredPermission, true)
}

func isUserAllowedTo(ctx context.Context, obj *models.ODObject, requiredPermission models.ODObjectPermission, rollup bool) (bool, models.ODObjectPermission) {
	caller, _ := CallerFromContext(ctx)
	groups, _ := GroupsFromContext(ctx)
	authorizedTo := false
	var userPermission models.ODObjectPermission
	var granteeMatch bool
	for _, permission := range obj.Permissions {
		if (!requiredPermission.AllowCreate || permission.AllowCreate) &&
			(!requiredPermission.AllowRead || permission.AllowRead) &&
			(!requiredPermission.AllowUpdate || permission.AllowUpdate) &&
			(!requiredPermission.AllowDelete || permission.AllowDelete) &&
			(!requiredPermission.AllowShare || permission.AllowShare) {
			granteeMatch = false
			if strings.Compare(strings.ToLower(permission.Grantee), strings.ToLower(caller.DistinguishedName)) == 0 {
				authorizedTo = true
				granteeMatch = true
			} else if strings.Compare(strings.ToLower(permission.Grantee), strings.ToLower(models.EveryoneGroup)) == 0 {
				authorizedTo = true
				granteeMatch = true
			} else {
				for _, group := range groups {
					if strings.Compare(strings.ToLower(permission.Grantee), strings.ToLower(group)) == 0 {
						authorizedTo = true
						granteeMatch = true
						break
					}
				}
			}
			if granteeMatch {
				if authorizedTo && !rollup {
					// short circuit, return the first matching permission
					return authorizedTo, permission
				}
				// we will examine all permissions in the set, building the combined
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
			}
		}
	}
	// Return the overall (either combined from one or more granteeMatch, or empty from no match)
	return authorizedTo, userPermission
}

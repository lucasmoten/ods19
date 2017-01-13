package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"fmt"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/utils"
)

func (h AppServer) updateObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, _ := CallerFromContext(ctx)
	logger := LoggerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MODIFY")

	aacAuth := auth.NewAACAuth(logger, h.AAC)

	if r.Header.Get("Content-Type") != "application/json" {
		herr := NewAppError(400, nil, "expected application/json Content-Type")
		h.publishError(gem, herr)
		return herr
	}

	requestObject, err = parseUpdateObjectRequestAsJSON(r, ctx)
	if err != nil {
		herr := NewAppError(400, err, fmt.Sprintf("Error parsing JSON %s", err.Error()))
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(500, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)

	dp := ciphertext.FindCiphertextCacheByObject(&dbObject)
	masterKey := dp.GetMasterKey()

	// Check if the user has permissions to update the ODObject
	var grant models.ODObjectPermission
	var ok bool
	if ok, grant = isUserAllowedToUpdateWithPermission(ctx, &dbObject); !ok {
		herr := NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
		h.publishError(gem, herr)
		return herr
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		herr := ClassifyObjectACMError(err)
		h.publishError(gem, herr)
		return herr
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			herr := NewAppError(410, nil, "The object no longer exists.")
			h.publishError(gem, herr)
			return herr
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			herr := NewAppError(405, nil, "The object cannot be modified because an ancestor is deleted.")
			h.publishError(gem, herr)
			return herr
		case dbObject.IsDeleted:
			herr := NewAppError(405, nil, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
			h.publishError(gem, herr)
			return herr
		}
	}

	// Check that assignment as deleted isn't occuring here. Should use deleteObject operations
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		herr := NewAppError(428, errors.New("Precondition required: Updating object as deleted not allowed. Send to trash or DELETE instead."), "Assigning object as deleted through update operation not allowed. Use deleteObject operation")
		h.publishError(gem, herr)
		return herr
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		herr := NewAppError(428, errors.New("Precondition required: ChangeToken does not match expected value"), "ChangeToken does not match expected value. Object may have been changed by another request.")
		h.publishError(gem, herr)
		return herr
	}

	// Retain existing value for parent.
	requestObject.ParentID = dbObject.ParentID

	// Retain existing values for content stream info
	requestObject.ContentConnector = dbObject.ContentConnector
	requestObject.ContentType = dbObject.ContentType
	requestObject.ContentSize = dbObject.ContentSize
	requestObject.ContentHash = dbObject.ContentHash
	requestObject.EncryptIV = dbObject.EncryptIV

	// Retain existing ownership
	requestObject.OwnedBy = models.ToNullString(dbObject.OwnedBy.String)

	// If there was no ACM provided...
	if len(requestObject.RawAcm.String) == 0 {
		// There was no change, retain existing from dbObject
		requestObject.RawAcm = models.ToNullString(dbObject.RawAcm.String)
	}

	// Assign existing permissions from the database object to the request object
	if len(requestObject.Permissions) == 0 {
		requestObject.Permissions = dbObject.Permissions
	} else {
		combinedPermissions := make([]models.ODObjectPermission, len(requestObject.Permissions)+len(dbObject.Permissions))
		// Any existing permissions will be marked as deleted, since past in overrides.
		idx := 0
		for _, d := range dbObject.Permissions {
			d.IsDeleted = true
			combinedPermissions[idx] = d
			idx = idx + 1
		}
		for _, r := range requestObject.Permissions {
			combinedPermissions[idx] = r
			idx = idx + 1
		}
		requestObject.Permissions = combinedPermissions
	}

	flattenedACM, err := aacAuth.GetFlattenedACM(requestObject.RawAcm.String)
	if err != nil {
		herr := ClassifyFlattenError(err)
		h.publishError(gem, herr)
		return herr

	}
	requestObject.RawAcm = models.ToNullString(flattenedACM)
	modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(requestObject.OwnedBy.String, requestObject.Permissions, requestObject.RawAcm.String, requestObject.IsCreating())
	if err != nil {
		herr := NewAppError(500, err, err.Error())
		h.publishError(gem, herr)
		return herr

	}
	requestObject.RawAcm = models.ToNullString(modifiedACM)
	requestObject.Permissions = modifiedPermissions
	// Access check against altered ACM as a whole
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, requestObject.RawAcm.String); err != nil {
		herr := ClassifyObjectACMError(err)
		h.publishError(gem, herr)
		return herr

	}
	consolidateChangingPermissions(&requestObject)
	// copy grant.EncryptKey to all existing permissions:
	for idx, permission := range requestObject.Permissions {
		models.CopyEncryptKey(masterKey, &grant, &permission)
		models.CopyEncryptKey(masterKey, &grant, &requestObject.Permissions[idx])
	}

	// If ACM provided differs from what is currently set, then need to
	// Check AAC to compare user clearance to NEW metadata Classifications
	// to see if allowed for this user
	if strings.Compare(dbObject.RawAcm.String, requestObject.RawAcm.String) != 0 {
		// Ensure user is allowed this acm
		if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, requestObject.RawAcm.String); err != nil {
			herr := ClassifyObjectACMError(err)
			h.publishError(gem, herr)
			return herr

		}
		// If the "share" or "f_share" parts have changed, then check that the
		// caller also has permission to share.
		if diff, herr := isAcmShareDifferent(dbObject.RawAcm.String, requestObject.RawAcm.String); herr != nil || diff {
			if herr != nil {
				h.publishError(gem, herr)
				return herr
			}
			// Need to refetch dbObject as apparently the assignment of its permissions into request object is a reference instead of copy
			dbPermissions, _ := dao.GetPermissionsForObject(dbObject)
			dbObject.Permissions = dbPermissions
			if !isUserAllowedToShare(ctx, &dbObject) {
				herr := NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to change the share for this object")
				h.publishError(gem, herr)
				return herr
			}
		}
	}

	// Retain existing values from dbObject where no value was provided for key fields
	if len(requestObject.Name) == 0 {
		requestObject.Name = dbObject.Name
	}
	if len(requestObject.Description.String) == 0 {
		requestObject.Description.String = dbObject.Description.String
	}
	if len(requestObject.TypeName.String) == 0 {
		requestObject.TypeName.String = dbObject.TypeName.String
	}
	if len(requestObject.ContainsUSPersonsData) == 0 {
		requestObject.ContainsUSPersonsData = dbObject.ContainsUSPersonsData
	}
	if len(requestObject.ExemptFromFOIA) == 0 {
		requestObject.ExemptFromFOIA = dbObject.ExemptFromFOIA
	}

	// Call metadata connector to update the object in the data store
	// Force the modified by to be that of the caller
	requestObject.ModifiedBy = caller.DistinguishedName
	err = dao.UpdateObject(&requestObject)
	if err != nil {
		herr := NewAppError(500, err, "DAO Error updating object")
		h.publishError(gem, herr)
		return herr
	}

	// After the update, check that key values have changed...
	if requestObject.ChangeCount <= dbObject.ChangeCount {
		logger.Error("ChangeCount didn't update when processing request",
			zap.Int("old", requestObject.ChangeCount), zap.Int("new", dbObject.ChangeCount),
			zap.String("requestObject.ID", hex.EncodeToString(requestObject.ID)), zap.String("dbObject.ID", hex.EncodeToString(dbObject.ID)))
		herr := NewAppError(500, nil, "ChangeCount didn't update when processing request")
		h.publishError(gem, herr)
		return herr

	}
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
		logger.Error("ChangeToken didn't update when procesing request",
			zap.String("old token", requestObject.ChangeToken), zap.String("new token", dbObject.ChangeToken))
		herr := NewAppError(500, nil, "ChangeToken didn't update when processing request")
		h.publishError(gem, herr)
		return herr
	}

	dbObject, err = dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(500, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditModified := NewResourceFromObject(dbObject)
	apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller))

	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
	gem.Payload.StreamUpdate = false
	h.publishSuccess(gem, r)

	jsonResponse(w, apiResponse)
	return nil
}

func parseUpdateObjectRequestAsJSON(r *http.Request, ctx context.Context) (models.ODObject, error) {
	var jsonObject protocol.UpdateObjectRequest
	requestObject := models.ODObject{}
	var err error

	// Get ID from URI
	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		return requestObject, errors.New("Object Identifier in Request URI is not a hex string")
	}

	// Get portion from body
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return requestObject, err
	}

	if strings.Compare(hex.EncodeToString(requestObject.ID), jsonObject.ID) != 0 {
		return requestObject, errors.New("bad request: ID mismatch")
	}

	// Map changes over the requestObject
	if len(jsonObject.Name) > 0 {
		if strings.IndexAny(jsonObject.Name, "/\\") > -1 {
			return requestObject, errors.New("bad request: name cannot include reserved characters {\\,/}")
		}
		requestObject.Name = jsonObject.Name
	}
	requestObject.ChangeToken = jsonObject.ChangeToken
	if len(jsonObject.TypeName) > 0 {
		requestObject.TypeName = models.ToNullString(jsonObject.TypeName)
	}
	if len(jsonObject.Description) > 0 {
		requestObject.Description = models.ToNullString(jsonObject.Description)
	}
	convertedAcm, err := utils.MarshalInterfaceToString(jsonObject.RawAcm)
	if err != nil {
		return requestObject, err
	}
	if len(convertedAcm) > 0 {
		requestObject.RawAcm = models.ToNullString(convertedAcm)
	}
	requestObject.Permissions, err = mapping.MapPermissionToODPermissions(&jsonObject.Permission)
	if err != nil {
		return requestObject, err
	}
	if len(jsonObject.ContainsUSPersonsData) > 0 {
		requestObject.ContainsUSPersonsData = jsonObject.ContainsUSPersonsData
	}
	if len(jsonObject.ExemptFromFOIA) > 0 {
		requestObject.ExemptFromFOIA = jsonObject.ExemptFromFOIA
	}
	if len(jsonObject.Properties) > 0 {
		requestObject.Properties, err = mapping.MapPropertiesToODProperties(&jsonObject.Properties)
	}

	// Return it
	return requestObject, err
}

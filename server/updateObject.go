package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"golang.org/x/net/context"

	"fmt"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/events"
	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/services/audit"
	"github.com/deciphernow/object-drive-server/util"
	"github.com/deciphernow/object-drive-server/utils"
)

func (h AppServer) updateObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error
	var recursive bool

	caller, _ := CallerFromContext(ctx)
	logger := LoggerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MODIFY")

	aacAuth := auth.NewAACAuth(logger, h.AAC)

	requestObject, recursive, err = parseUpdateObjectRequestAsJSON(ctx, r)
	if err != nil {
		herr := NewAppError(400, err, err.Error())
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	// Retrieve existing object from the database.
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(500, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)

	// Check if the user has permissions to update the ODObject
	var grant models.ODObjectPermission
	var ok bool

	if ok, grant = isUserAllowedToUpdateWithPermission(ctx, &dbObject); !ok {
		herr := NewAppError(403, errors.New("forbidden"), "user does not have permission to update this object")
		h.publishError(gem, herr)
		return herr
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		herr := NewAppError(authHTTPErr(err), err, err.Error())
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
		herr := NewAppError(428, errors.New("updating object as deleted not allowed"), "Assigning object as deleted through update operation not allowed. Use deleteObject operation")
		h.publishError(gem, herr)
		return herr
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		herr := NewAppError(428, errors.New("changeToken does not match expected value"), "changeToken does not match expected value")
		h.publishError(gem, herr)
		return herr
	}

	// Retain existing value for parent.
	requestObject.ParentID = dbObject.ParentID

	// Retain existing values for content stream info
	requestObject.ContentConnector = dbObject.ContentConnector
	if !requestObject.ContentType.Valid {
		// only if not set from the update
		requestObject.ContentType = dbObject.ContentType
	}
	requestObject.ContentSize = dbObject.ContentSize
	requestObject.ContentHash = dbObject.ContentHash
	requestObject.EncryptIV = dbObject.EncryptIV

	// Retain existing ownership
	requestObject.OwnedBy = models.ToNullString(dbObject.OwnedBy.String)

	updatingAcm := len(requestObject.RawAcm.String) > 0
	updatingPermissions := len(requestObject.Permissions) > 0
	if !updatingAcm {
		// start with existing acm from database
		requestObject.RawAcm = models.ToNullString(dbObject.RawAcm.String)
	}
	if !updatingPermissions {
		// use existing permissions from database
		requestObject.Permissions = dbObject.Permissions
		if updatingAcm {
			// existing permissions will be marked as deleted, acm is authoritative
			for pidx := range requestObject.Permissions {
				requestObject.Permissions[pidx].IsDeleted = true
			}
		}
	} else {
		combinedPermissions := make([]models.ODObjectPermission, len(requestObject.Permissions)+len(dbObject.Permissions))
		idx := 0
		// existing permissions will be marked as deleted
		for _, d := range dbObject.Permissions {
			d.IsDeleted = true
			combinedPermissions[idx] = d
			idx = idx + 1
		}
		// passed in explicitly overrides
		for _, r := range requestObject.Permissions {
			combinedPermissions[idx] = r
			idx = idx + 1
		}
		requestObject.Permissions = combinedPermissions
	}

	modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(requestObject.OwnedBy.String, requestObject.Permissions, requestObject.RawAcm.String, requestObject.IsCreating())
	if err != nil {
		herr := NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return herr

	}
	requestObject.RawAcm = models.ToNullString(modifiedACM)
	requestObject.Permissions = modifiedPermissions
	// Access check against altered ACM as a whole
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, requestObject.RawAcm.String); err != nil {
		herr := NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return herr

	}
	consolidateChangingPermissions(&requestObject)

	masterKey := ciphertext.FindCiphertextCacheByObject(nil).GetMasterKey()
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
			herr := NewAppError(authHTTPErr(err), err, err.Error())
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
	gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)

	if recursive {

		applyable := dbObject
		go h.updateObjectRecursive(ctx, applyable)

	}
	return nil
}

func (h AppServer) updateObjectRecursive(ctx context.Context, applyable models.ODObject) {
	d := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "SHARE_MODIFY")

	page := 1
	pr := dao.PagingRequest{PageNumber: page, PageSize: dao.MaxPageSize}

	children, err := d.GetChildObjectsWithProperties(pr, applyable)
	if err != nil {
		logger.Error("error calling GetChildObjectsWithProperties", zap.Error(err))
		return
	}

	for {
		for _, child := range children.Objects {

			gem.Payload.ObjectID = hex.EncodeToString(child.ID)
			gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(child.ID))
			auditOriginal := NewResourceFromObject(child)

			if child.IsDeleted {
				continue
			}

			ok, updatePermission := isUserAllowedToShareWithPermission(ctx, &child)
			if !ok {
				herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
				h.publishError(gem, herr)
				continue
			}

			if updatePermission.AcmGrantee.Grantee == "" {
				logger.Error("grantee cannot be empty string", zap.Error(err))
				gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
				h.EventQueue.Publish(gem)
				continue
			}

			// newACM, err := aacAuth.InjectPermissionsIntoACM(applyable.Permissions, child.RawAcm.String)
			newACM, err := aacAuth.InjectPermissionsIntoACM(applyable.Permissions, child.RawAcm.String)
			if err != nil {
				gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
				h.EventQueue.Publish(gem)
				continue
			}
			// newPerms, newACM2, err := aacAuth.NormalizePermissionsFromACM(child.OwnedBy.String, child.Permissions, newACM, false)
			newPerms, newACM2, err := aacAuth.NormalizePermissionsFromACM(child.OwnedBy.String, applyable.Permissions, newACM, false)
			if err != nil {
				gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
				h.EventQueue.Publish(gem)
				continue
			}
			child.Permissions = newPerms

			newerACM, err := aacAuth.RebuildACMFromPermissions(child.Permissions, newACM2)
			if err != nil {
				gem.Payload.Audit = audit.WithActionResult(gem.Payload.Audit, "FAILURE")
				h.EventQueue.Publish(gem)
				continue
			}
			child.RawAcm = models.ToNullString(newerACM)

			if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, child.RawAcm.String); err != nil {
				logger.Error("error calling IsUserAuthorizedForACM", zap.Error(err))
				continue
			}
			consolidateChangingPermissions(&child)
			// Get around: Invalid MAC on permission
			masterKey := ciphertext.FindCiphertextCacheByObject(nil).GetMasterKey()
			for i, p := range child.Permissions {
				models.CopyEncryptKey(masterKey, &updatePermission, &p)
				models.CopyEncryptKey(masterKey, &updatePermission, &child.Permissions[i])
			}
			child.ModifiedBy = caller.DistinguishedName
			err = d.UpdateObject(&child)
			if err != nil {
				logger.Error("error updating child object with new permissions", zap.Error(err))
				continue
			}

			auditModified := NewResourceFromObject(child)
			gem.Payload.Audit = audit.WithModifiedPairList(
				gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
			apiResponse := mapping.MapODObjectToObject(&child)
			gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
			h.EventQueue.Publish(gem)
			h.updateObjectRecursive(ctx, child)
		}
		page++
		if page > children.PageCount {
			break
		}
		pr.PageNumber = page
		var err error
		children, err = d.GetChildObjectsWithProperties(pr, applyable)
		if err != nil {
			logger.Error("error calling GetChildObjectsWithProperties", zap.Error(err))
			return
		}
	}
}

// parseUpdateObjectRequestAsJSON parses a request into our models object.
// Internally the function inspects HTTP headers, URL params, and decodes
// the request's JSON body. Parsed data is mapped into the returned models.ODObject type.
//
// TODO(cm): We delegate to 2 custom mapping funcs in this function. This function is
// a constructor in disguise.
func parseUpdateObjectRequestAsJSON(ctx context.Context, r *http.Request) (models.ODObject, bool, error) {
	var jsonObject protocol.UpdateObjectRequest
	var requestObject models.ODObject
	var err error
	var recursive bool

	if !util.IsApplicationJSON(r.Header.Get("Content-Type")) {
		return requestObject, false, errors.New("expected Content-Type: application/json")
	}

	// Instantiate models.ODObject with []byte ID set from URL capture groups.
	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		return requestObject, false, errors.New("Object Identifier in Request URI is not a hex string")
	}

	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return requestObject, false, err
	}
	recursive = jsonObject.RecursiveShare

	if strings.Compare(hex.EncodeToString(requestObject.ID), jsonObject.ID) != 0 {
		return requestObject, false, errors.New("bad request: ID mismatch")
	}

	// Map changes over the requestObject
	if jsonObject.Name != "" {
		if part, _ := util.GetNextDelimitedPart(jsonObject.Name, util.DefaultPathDelimiter); len(part) > 0 {
			return requestObject, false, fmt.Errorf("bad request: name cannot include path delimiter %s", util.DefaultPathDelimiter)
		}
		requestObject.Name = jsonObject.Name
	}
	requestObject.ChangeToken = jsonObject.ChangeToken
	requestObject.TypeName = models.ToNullString(jsonObject.TypeName)
	requestObject.Description = models.ToNullString(jsonObject.Description)

	convertedAcm, err := utils.MarshalInterfaceToString(jsonObject.RawAcm)
	if err != nil {
		return requestObject, false, err
	}
	requestObject.RawAcm = models.ToNullString(convertedAcm)
	requestObject.Permissions, err = mapping.MapPermissionToODPermissions(&jsonObject.Permission)
	if err != nil {
		return requestObject, false, err
	}
	requestObject.ContentType = models.ToNullString(jsonObject.ContentType)
	requestObject.ContainsUSPersonsData = jsonObject.ContainsUSPersonsData
	requestObject.ExemptFromFOIA = jsonObject.ExemptFromFOIA
	if len(jsonObject.Properties) > 0 {
		requestObject.Properties, err = mapping.MapPropertiesToODProperties(&jsonObject.Properties)
	}

	// Return it
	return requestObject, recursive, err
}

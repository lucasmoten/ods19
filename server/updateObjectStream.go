package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/crypto"
	db "github.com/deciphernow/object-drive-server/dao"
	"github.com/deciphernow/object-drive-server/events"
	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) updateObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	var drainFunc func()
	var requestObject models.ODObject
	var err error
	var recursive bool

	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	captured, _ := CaptureGroupsFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)

	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MODIFY")

	if captured["objectId"] == "" {
		return NewAppError(http.StatusBadRequest, errors.New("bad request"), "invalid objectId in URI")
	}

	bytesID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return NewAppError(http.StatusBadRequest, errors.New("bad request"), "")
	}
	requestObject.ID = bytesID

	gem.Payload.ObjectID = captured["objectId"]
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		if err.Error() == db.ErrNoRows.Error() {
			herr := NewAppError(404, err, "Not found")
			h.publishError(gem, herr)
			return herr
		}
		herr := NewAppError(500, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)

	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			herr := NewAppError(410, err, "The object no longer exists.")
			h.publishError(gem, herr)
			return herr
		case dbObject.IsAncestorDeleted:
			herr := NewAppError(405, err, "The object cannot be modified because an ancestor is deleted.")
			h.publishError(gem, herr)
			return herr
		default:
			herr := NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
			h.publishError(gem, herr)
			return herr
		}
	}

	// We need a name for the new text, and a new iv
	dbObject.ContentConnector.String = crypto.CreateRandomName()
	dbObject.EncryptIV = crypto.CreateIV()
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
		herr := NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return herr
	}

	multipartReader, err := r.MultipartReader()
	if err != nil {
		herr := NewAppError(400, err, "unable to open multipart reader")
		h.publishError(gem, herr)
		return herr
	}
	drainFunc, _, recursive, herr := h.acceptObjectUpload(ctx, multipartReader, &dbObject, &grant, false, nil)
	dp := ciphertext.FindCiphertextCacheByObject(&dbObject)
	if herr != nil {
		herr := abortUploadObject(logger, dp, &dbObject, true, herr)
		h.publishError(gem, herr)
		return herr
	}
	masterKey := dp.GetMasterKey()
	modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(dbObject.OwnedBy.String, dbObject.Permissions, dbObject.RawAcm.String, dbObject.IsCreating())
	if err != nil {
		herr = NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	dbObject.RawAcm = models.ToNullString(modifiedACM)
	dbObject.Permissions = modifiedPermissions
	// Final access check against altered ACM
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		herr = NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}

	// Verify user has access to change the share if the ACMs are different
	unchangedDbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr = NewAppError(500, err, err.Error())
		h.publishError(gem, herr)
		return herr
	}
	// If the "share" or "f_share" parts have changed, then check that the
	// caller also has permission to share.
	if diff, herr := isAcmShareDifferent(dbObject.RawAcm.String, unchangedDbObject.RawAcm.String); herr != nil || diff {
		if herr != nil {
			h.publishError(gem, herr)
			return abortUploadObject(logger, dp, &dbObject, true, herr)
		}
		if !isUserAllowedToShare(ctx, &unchangedDbObject) {
			herr = NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to change the share for this object")
			h.publishError(gem, herr)
			return abortUploadObject(logger, dp, &dbObject, true, herr)
		}
	}

	consolidateChangingPermissions(&dbObject)
	// copy grant.EncryptKey to all existing permissions:
	for idx, permission := range dbObject.Permissions {
		models.CopyEncryptKey(masterKey, &grant, &permission)
		models.CopyEncryptKey(masterKey, &grant, &dbObject.Permissions[idx])
	}

	dbObject.ModifiedBy = caller.DistinguishedName
	err = dao.UpdateObject(&dbObject)
	if err != nil {
		herr = NewAppError(500, err, "error storing object")
		h.publishError(gem, herr)
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	parents, err := dao.GetParents(dbObject)
	if err != nil {
		herr := NewAppError(500, err, "error retrieving object parents")
		h.publishError(gem, herr)
		return herr
	}

	filtered := redactParents(ctx, aacAuth, parents)
	if appError := errOnDeletedParents(parents); appError != nil {
		h.publishError(gem, appError)
		return appError
	}
	crumbs := breadcrumbsFromParents(filtered)

	auditModified := NewResourceFromObject(dbObject)
	// Only start to upload into S3 after we have a database record
	go drainFunc()

	apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller)).WithBreadcrumbs(crumbs)

	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.StreamUpdate = false
	gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
	gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)

	if recursive {
		go h.updateObjectRecursive(ctx, dbObject)
	}
	return nil
}

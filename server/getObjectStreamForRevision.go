package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/crypto"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/services/audit"
)

func (h AppServer) getObjectStreamForRevision(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	var requestObject models.ODObject
	var err error
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		herr := NewAppError(500, errors.New("Could not get capture groups"), "No capture groups.")
		h.publishError(gem, herr)
		return herr
	}

	if captured["objectId"] == "" {
		herr := NewAppError(http.StatusBadRequest, errors.New("Could not extract objectID from URI"), "URI: "+r.URL.Path)
		h.publishError(gem, herr)
		return herr
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Invalid objectID in URI.")
		h.publishError(gem, herr)
		return herr
	}
	requestObject.ID = bytesObjectID
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	if captured["revisionId"] == "" {
		herr := NewAppError(http.StatusBadRequest, errors.New("Could not extract revisionId from URI"), "URI: "+r.URL.Path)
		h.publishError(gem, herr)
		return herr
	}
	requestObject.ChangeCount, err = strconv.Atoi(captured["revisionId"])
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Invalid revisionId in URI.")
		h.publishError(gem, herr)
		return herr
	}
	var fileKey []byte
	var herr *AppError
	// Current version authorization checks
	dbObjectCurrent, err := dao.GetObject(requestObject, false)
	if err != nil {
		herr := NewAppError(500, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	if herr, fileKey = getFileKeyAndCheckAuthAndObjectState(ctx, h, &dbObjectCurrent); herr != nil {
		h.publishError(gem, herr)
		return herr
	}
	// Requested revision
	dbObjectRevision, err := dao.GetObjectRevision(requestObject, true)
	if err != nil {
		herr := NewAppError(500, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(dbObjectRevision))
	gem.Payload.ChangeToken = dbObjectRevision.ChangeToken
	if herr, fileKey = getFileKeyAndCheckAuthAndObjectState(ctx, h, &dbObjectRevision); herr != nil {
		h.publishError(gem, herr)
		return herr
	}

	// Fail fast: Don't even look at cache or retrieve if the file size is 0
	if !dbObjectRevision.ContentSize.Valid || dbObjectRevision.ContentSize.Int64 <= int64(0) {
		herr := NewAppError(204, nil, "No content")
		h.publishSuccess(gem, w)
		return herr
	}

	disposition := "inline"
	overrideDisposition := r.URL.Query().Get("disposition")
	if len(overrideDisposition) > 0 {
		disposition = overrideDisposition
	}
	ctx = ContextWithGEM(ctx, gem)
	_, appError := h.getAndStreamFile(ctx, &dbObjectRevision, w, r, fileKey, false, disposition)
	if appError != nil {
		if appError.Error != nil {
			h.publishError(gem, appError)
		} else {
			h.publishSuccess(gem, w)
		}
		return appError
	}
	h.publishSuccess(gem, w)
	return nil
}

func getFileKeyAndCheckAuthAndObjectState(ctx context.Context, h AppServer, dbObject *models.ODObject) (*AppError, []byte) {
	var fileKey []byte

	ok, userPermission := isUserAllowedToReadWithPermission(ctx, dbObject)
	if !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to read/view this object"), fileKey
	}

	dp := ciphertext.FindCiphertextCacheByObject(dbObject)
	masterKey := dp.GetMasterKey()

	fileKey = crypto.ApplyPassphrase(masterKey, userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		return NewAppError(500, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object"), fileKey
	}

	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		return ClassifyObjectACMError(err), fileKey
	}

	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, nil, "The object no longer exists."), fileKey
		case dbObject.IsAncestorDeleted:
			return NewAppError(405, nil, "The object cannot be retreived because an ancestor is deleted."), fileKey
		default:
			return NewAppError(405, nil, "The object is deleted"), fileKey
		}
	}

	return nil, fileKey
}

package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/crypto"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) getObjectStreamForRevision(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)

	var requestObject models.ODObject
	var err error

	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not get capture groups"), "No capture groups.")
	}

	if captured["objectId"] == "" {
		return NewAppError(http.StatusBadRequest, errors.New("Could not extract objectID from URI"), "URI: "+r.URL.Path)
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return NewAppError(http.StatusBadRequest, err, "Invalid objectID in URI.")
	}
	requestObject.ID = bytesObjectID
	if captured["revisionId"] == "" {
		return NewAppError(http.StatusBadRequest, errors.New("Could not extract revisionId from URI"), "URI: "+r.URL.Path)
	}
	requestObject.ChangeCount, err = strconv.Atoi(captured["revisionId"])
	if err != nil {
		return NewAppError(http.StatusBadRequest, err, "Invalid revisionId in URI.")
	}
	var fileKey []byte
	var herr *AppError
	// Current version authorization checks
	dbObjectCurrent, err := dao.GetObject(requestObject, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}
	if herr, fileKey = getFileKeyAndCheckAuthAndObjectState(ctx, h, &dbObjectCurrent); herr != nil {
		return herr
	}
	// Requested revision
	dbObjectRevision, err := dao.GetObjectRevision(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}
	if herr, fileKey = getFileKeyAndCheckAuthAndObjectState(ctx, h, &dbObjectRevision); herr != nil {
		return herr
	}

	// Fail fast: Don't even look at cache or retrieve if the file size is 0
	if !dbObjectRevision.ContentSize.Valid || dbObjectRevision.ContentSize.Int64 <= int64(0) {
		return NewAppError(204, nil, "No content")
	}

	disposition := "inline"
	overrideDisposition := r.URL.Query().Get("disposition")
	if len(overrideDisposition) > 0 {
		disposition = overrideDisposition
	}

	_, appError := h.getAndStreamFile(ctx, &dbObjectRevision, w, r, fileKey, false, disposition)
	if appError != nil {
		return appError
	}
	return nil
}

func getFileKeyAndCheckAuthAndObjectState(ctx context.Context, h AppServer, obj *models.ODObject) (*AppError, []byte) {
	var fileKey []byte

	ok, userPermission := isUserAllowedToReadWithPermission(ctx, h.MasterKey, obj)
	if !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to read/view this object"), fileKey
	}
	fileKey = crypto.ApplyPassphrase(h.MasterKey, userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		return NewAppError(500, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object"), fileKey
	}

	if err := h.isUserAllowedForObjectACM(ctx, obj); err != nil {
		return ClassifyObjectACMError(err), fileKey
	}

	if obj.IsDeleted {
		switch {
		case obj.IsExpunged:
			return NewAppError(410, nil, "The object no longer exists."), fileKey
		case obj.IsAncestorDeleted:
			return NewAppError(405, nil, "The object cannot be retreived because an ancestor is deleted."), fileKey
		default:
			return NewAppError(405, nil, "The object is deleted"), fileKey
		}
	}

	return nil, fileKey
}

package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) getObjectStreamForRevision(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// // Get caller value from ctx.
	// caller, ok := CallerFromContext(ctx)
	// if !ok {
	// 	return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	// }
	dao := DAOFromContext(ctx)

	var requestObject models.ODObject
	var err error

	// Parse the objectId and historyId (changeCount) from the request path

	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not get capture groups"), "No capture groups.")
	}

	// Initialize requestobject with the objectId being requested
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

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObjectRevision(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Check read permission, and capture permission for the encryptKey
	ok, userPermission := isUserAllowedToReadWithPermission(ctx, h.MasterKey, &dbObject)
	if !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to read/view this object")
	}
	// Using captured permission, derive filekey
	var fileKey []byte
	fileKey = utils.ApplyPassphrase(h.MasterKey, userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		return NewAppError(500, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object")
	}

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, err, "Forbidden - User does not pass authorization checks for object ACM")
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted:
			return NewAppError(405, err, "The object cannot be retreived because an ancestor is deleted.")
		default:
			return NewAppError(405, err, "The object is deleted")
		}
	}

	// Fail fast: Don't even look at cache or retrieve if the file size is 0
	if !dbObject.ContentSize.Valid || dbObject.ContentSize.Int64 <= int64(0) {
		return NewAppError(204, nil, "No content")
	}

	disposition := "inline"
	overrideDisposition := r.URL.Query().Get("disposition")
	if len(overrideDisposition) > 0 {
		disposition = overrideDisposition
	}

	_, appError := h.getAndStreamFile(ctx, &dbObject, w, r, fileKey, false, disposition)
	if appError != nil {
		return appError
	}
	return nil
}

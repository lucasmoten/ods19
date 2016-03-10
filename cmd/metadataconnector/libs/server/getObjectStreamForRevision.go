package server

import (
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/util"
)

func (h AppServer) getObjectStreamForRevision(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	var requestObject models.ODObject
	var err error

	// Parse the objectId and historyId (changeCount) from the request path
	captured := util.GetRegexCaptureGroups(r.URL.Path, h.Routes.ObjectStreamRevision)
	if captured["objectId"] == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, errors.New("Could not extract objectID from URI"), "URI: "+r.URL.Path)
		return
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, err, "Invalid objectID in URI.")
	}
	requestObject.ID = bytesObjectID
	if captured["historyId"] == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, errors.New("Could not extract historyID from URI"), "URI: "+r.URL.Path)
		return
	}
	requestObject.ChangeCount, err = strconv.Atoi(captured["historyId"])
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, err, "Invalid historyID in URI.")
	}

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObjectRevision(requestObject, true)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object")
		return
	}

	// Check if the user has permissions to read the ODObject
	//		Permission.grantee matches caller, and AllowRead is true
	authorizedToRead := false
	var userEncryptKey []byte
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowRead {
			authorizedToRead = true
			userEncryptKey = permission.EncryptKey
			//Unscramble the Key with the masterkey
			utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, userEncryptKey)
		}
	}
	if !authorizedToRead {
		log.Printf("Failed Permission check")
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error communicating with authorization service")
		return
	}
	if !hasAACAccess {
		log.Printf("Failed ACM check")
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted:
			h.sendErrorResponse(w, 405, err, "The object cannot be retreived because an ancestor is deleted.")
			return
		default:
			h.sendErrorResponse(w, 405, err, "The object is deleted")
			return
		}
	}

	// Fail fast: Don't even look at cache or retrieve if the file size is 0
	if !dbObject.ContentSize.Valid || dbObject.ContentSize.Int64 <= int64(0) {
		h.sendErrorResponse(w, 204, nil, "No content")
		return
	}

	// Fetch the stream for this object, sent to response writer
	appError := h.getAndStreamFile(ctx, &dbObject, w, userEncryptKey, false)
	if appError != nil {
		h.sendErrorResponse(w, appError.Code, appError.Err, appError.Msg)
		return
	}

}

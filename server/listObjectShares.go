package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) listObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	var err error

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, captured, true)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Fetch the matching object
	targetObject := models.ODObject{}
	targetObject.ID, _ = hex.DecodeString(pagingRequest.ObjectID)
	dbObject, err := dao.GetObject(targetObject, false)
	if err != nil {
		return NewAppError(404, err, "Resource not found")
	}

	// Check for permission to read this object
	if ok := isUserAllowedToShare(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to list shares of this object")
	}

	// Is it deleted?
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return NewAppError(405, err, "The object cannot be read because an ancestor is deleted.")
		case dbObject.IsDeleted:
			return NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its shares")
		}
	}

	// Response in requested format
	apiResponse := mapping.MapODPermissionsToPermissions(&dbObject.Permissions)
	jsonResponse(w, apiResponse)
	return nil
}

package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) listObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}
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
	canReadObject := false

	if strings.Compare(dbObject.OwnedBy.String, user.DistinguishedName) == 0 {
		canReadObject = true
	} else {
		for _, perm := range dbObject.Permissions {
			if perm.AllowRead && perm.AllowShare && perm.Grantee == user.DistinguishedName {
				canReadObject = true
				break
			}
		}
	}

	if !canReadObject {
		return NewAppError(403, err, "Insufficient permissions to view shares of this object")
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

	if err != nil {
		return NewAppError(500, err, "General error")
	}

	// Response in requested format
	return listObjectSharesResponseAsJSON(LoggerFromContext(ctx), w, mapping.MapODPermissionsToPermissions(&dbObject.Permissions))
}

func listObjectSharesResponseAsJSON(
	logger zap.Logger,
	w http.ResponseWriter,
	response []protocol.Permission,
) *AppError {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return NewAppError(500, err, "error marshalling json response")
	}
	w.Write(jsonData)
	return nil
}

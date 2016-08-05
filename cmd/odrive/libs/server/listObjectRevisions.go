package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// listObjectRevisions is a method handler on AppServer for implementing the
// listObjectRevisions microservice operation.
func (h AppServer) listObjectRevisions(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

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

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, captured, true)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Fetch matching object
	obj := models.ODObject{}
	// valid decoding checked when parsed, no need to check for error again
	obj.ID, err = hex.DecodeString(pagingRequest.ObjectID)
	if err != nil {
		return NewAppError(400, err, "Object Identifier in Request URI is not a hex string")
	}
	dbObject, err := dao.GetObject(obj, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Check for permission to read this object
	if ok := isUserAllowedToRead(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to list revisions of this object")
	}

	// Is it deleted?
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return NewAppError(405, err, "The object cannot be read because an ancestor is deleted.")
		case dbObject.IsDeleted:
			return NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its contents")
		}
	}

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		return NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
	}
	user.Snippets = snippetFields

	// Get the revision information for this objects
	response, err := dao.GetObjectRevisionsWithPropertiesByUser(user, *pagingRequest, dbObject)
	if err != nil {
		return NewAppError(500, err, "General error")
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
	jsonResponse(w, apiResponse)
	return nil
}

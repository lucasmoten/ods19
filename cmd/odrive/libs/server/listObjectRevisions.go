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
	dbObject, err := h.DAO.GetObject(obj, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Check for permission to read this object
	canReadObject := false
	for _, perm := range dbObject.Permissions {
		if perm.AllowRead && perm.Grantee == user.DistinguishedName {
			canReadObject = true
			break
		}
	}
	if !canReadObject {
		return NewAppError(403, err, "Insufficient permissions to list contents of this object")
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
	snippetFields, err := h.FetchUserSnippets(ctx)
	if err != nil {
		return NewAppError(504, errors.New("Error retrieving user permissions."), err.Error())
	}
	user.Snippets = snippetFields

	// Get the revision information for this objects
	response, err := h.DAO.GetObjectRevisionsWithPropertiesByUser(user, *pagingRequest, dbObject)
	if err != nil {
		return NewAppError(500, err, "General error")
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
	writeResultsetAsJSON(w, &apiResponse)
	return nil
}

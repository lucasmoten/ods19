package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"
)

// listObjectRevisions is a method handler on AppServer for implementing the
// listObjectRevisions microservice operation.
func (h AppServer) listObjectRevisions(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, captured, true)
	if err != nil {
		herr := NewAppError(400, err, "Error parsing request")
		h.publishError(gem, herr)
		return herr
	}

	// Fetch matching object
	obj := models.ODObject{}
	// valid decoding checked when parsed, no need to check for error again
	obj.ID, err = hex.DecodeString(pagingRequest.ObjectID)
	if err != nil {
		herr := NewAppError(400, err, "Object Identifier in Request URI is not a hex string")
		h.publishError(gem, herr)
		return herr
	}
	dbObject, err := dao.GetObject(obj, false)
	if err != nil {
		herr := NewAppError(500, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}

	// Check for permission to read this object
	if ok := isUserAllowedToRead(ctx, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to list revisions of this object")
	}

	// Is it deleted?
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			herr := NewAppError(410, err, "The object no longer exists.")
			h.publishError(gem, herr)
			return herr
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			herr := NewAppError(405, err, "The object cannot be read because an ancestor is deleted.")
			h.publishError(gem, herr)
			return herr
		case dbObject.IsDeleted:
			herr := NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its contents")
			h.publishError(gem, herr)
			return herr
		}
	}

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		herr := NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
		h.publishError(gem, herr)
		return herr
	}
	user.Snippets = snippetFields
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	checkACM := func(o *models.ODObject) bool {
		isAllowed, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, o.RawAcm.String)
		if err != nil {
			return false
		}
		return isAllowed
	}

	// Get the revision information for this objects
	response, err := dao.GetObjectRevisionsWithPropertiesByUser(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest), dbObject, checkACM)
	if err != nil {
		herr := NewAppError(500, err, "General error")
		h.publishError(gem, herr)
		return herr
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)

	// Caller permissions
	for objectIndex, object := range apiResponse.Objects {
		apiResponse.Objects[objectIndex] = object.WithCallerPermission(protocolCaller(caller))
	}

	gem.Payload.Audit = WithResourcesFromResultset(gem.Payload.Audit, response)

	// Output as JSON
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

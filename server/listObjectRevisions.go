package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
)

// listObjectRevisions is a method handler on AppServer for implementing the
// listObjectRevisions microservice operation.
func (h AppServer) listObjectRevisions(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)
	logger := LoggerFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "list"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, captured, true)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing request")
		h.publishError(gem, herr)
		return herr
	}

	// Fetch matching object
	obj := models.ODObject{}
	// valid decoding checked when parsed, no need to check for error again
	obj.ID, err = hex.DecodeString(pagingRequest.ObjectID)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Object Identifier in Request URI is not a hex string")
		h.publishError(gem, herr)
		return herr
	}
	dbObject, err := dao.GetObject(obj, false)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}

	// Check for permission to read this object
	if ok := isUserAllowedToRead(ctx, &dbObject); !ok {
		return NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to list revisions of this object")
	}

	// Is it deleted?
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			herr := NewAppError(http.StatusGone, err, "The object no longer exists.")
			h.publishError(gem, herr)
			return herr
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			herr := NewAppError(http.StatusConflict, err, "The object cannot be read because an ancestor is deleted.")
			h.publishError(gem, herr)
			return herr
		case dbObject.IsDeleted:
			herr := NewAppError(http.StatusConflict, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its contents")
			h.publishError(gem, herr)
			return herr
		}
	}

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusBadGateway, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
		h.publishError(gem, herr)
		return herr
	}
	user.Snippets = snippetFields

	// Get the revision information for this objects
	response, err := dao.GetObjectRevisionsByUser(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest), dbObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "General error")
		h.publishError(gem, herr)
		return herr
	}

	// Redact as appropriate
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	for i, o := range response.Objects {
		if isAllowed, _ := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, o.RawAcm.String); !isAllowed {
			ro := models.ODObject{
				ID:           o.ID,
				ChangeCount:  o.ChangeCount,
				CreatedBy:    o.CreatedBy,
				ModifiedBy:   o.ModifiedBy,
				CreatedDate:  o.CreatedDate,
				ModifiedDate: o.ModifiedDate,
			}
			response.Objects[i] = ro
		}
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

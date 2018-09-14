package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"bitbucket.di2e.net/dime/object-drive-server/events"
	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

func (h AppServer) deleteObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, errors.New("Could not get caller from context"), "Invalid caller.")
	}
	gem, _ := GEMFromContext(ctx)
	gem.Action = "delete"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventDelete")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "REMOVE")

	user, ok := UserFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusInternalServerError, errors.New("Could not determine user"), "Invalid user.")
		h.publishError(gem, herr)
		return herr
	}
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusBadGateway, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
		h.publishError(gem, herr)
		return herr
	}
	user.Snippets = snippetFields
	dao := DAOFromContext(ctx)

	// Get object
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	dbObject, err := dao.GetObject(requestObject, false)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(dbObject))
	gem.Payload.ChangeToken = dbObject.ChangeToken

	// Auth check
	if ok := isUserAllowedToDelete(ctx, &dbObject); !ok {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to delete this object")
		h.publishError(gem, herr)
		return herr
	}

	// State check
	if dbObject.IsDeleted {
		// Deleted already
		switch {
		case dbObject.IsExpunged:
			herr := NewAppError(http.StatusGone, err, "The referenced object no longer exists.")
			h.publishError(gem, herr)
			return herr
		default:
		}
	} else {
		// ok to change
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ChangeToken = requestObject.ChangeToken
		err = dao.DeleteObject(user, dbObject, true)
		if err != nil {
			herr := NewAppError(http.StatusInternalServerError, err, "DAO Error deleting object")
			h.publishError(gem, herr)
			return herr
		}
	}

	// reget the object so that changetoken and deleteddate are correct
	dbObject, err = dao.GetObject(requestObject, false)

	// Response in requested format
	apiResponse := mapping.MapODObjectToDeletedObjectResponse(&dbObject).WithCallerPermission(protocolCaller(caller))
	gem.Payload.StreamUpdate = false
	gem.Payload = events.WithEnrichedPayload(gem.Payload, mapping.MapODObjectToObject(&dbObject))
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

// This same handler is used for both deleting an object (POST as new state), or deleting forever (DELETE)
func parseDeleteObjectRequest(r *http.Request, ctx context.Context) (models.ODObject, error) {
	var jsonObject protocol.DeleteObjectRequest
	var requestObject models.ODObject
	var err error

	// Depends on this for the changeToken
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return requestObject, err
	}

	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return requestObject, errors.New("Could not get capture groups")
	}

	// Initialize requestobject with the objectId being requested
	if captured["objectId"] == "" {
		return requestObject, errors.New("Could not extract ObjectID from URI")
	}
	_, err = hex.DecodeString(captured["objectId"])
	if err != nil {
		return requestObject, errors.New("Invalid ObjectID in URI")
	}
	jsonObject.ID = captured["objectId"]

	// Map to internal object type
	requestObject, err = mapping.MapDeleteObjectRequestToODObject(&jsonObject)
	return requestObject, err
}

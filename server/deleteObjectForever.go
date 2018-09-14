package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"bitbucket.di2e.net/dime/object-drive-server/events"
	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
)

func (h AppServer) deleteObjectForever(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, errors.New("Could not get caller from context"), "Invalid caller.")
	}
	gem, _ := GEMFromContext(ctx)
	gem.Action = "delete"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventDelete")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "DELETE")

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

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(dbObject))
	gem.Payload.ChangeToken = dbObject.ChangeToken

	// Auth check
	if ok := isUserAllowedToDelete(ctx, &dbObject); !ok {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to expunge this object")
		h.publishError(gem, herr)
		return herr
	}

	// Check state
	if dbObject.IsExpunged {
		herr := NewAppError(http.StatusGone, err, "The referenced object no longer exists.")
		h.publishError(gem, herr)
		return herr
	}

	dbObject.ModifiedBy = caller.DistinguishedName
	dbObject.ChangeToken = requestObject.ChangeToken
	err = dao.ExpungeObject(user, dbObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "DAO Error expunging object")
		h.publishError(gem, herr)
		return herr
	}

	apiResponse := mapping.MapODObjectToExpungedObjectResponse(&dbObject).WithCallerPermission(protocolCaller(caller))
	jsonResponse(w, apiResponse)
	gem.Payload = events.WithEnrichedPayload(gem.Payload, mapping.MapODObjectToObject(&dbObject))
	h.publishSuccess(gem, w)
	return nil
}

package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) deleteObjectForever(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, errors.New("Could not get caller from context"), "Invalid caller.")
	}
	gem, _ := GEMFromContext(ctx)
	session := SessionIDFromContext(ctx)
	user, ok := UserFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, errors.New("Could not determine user"), "Invalid user.")
	}
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusBadGateway, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
	}
	user.Snippets = snippetFields
	dao := DAOFromContext(ctx)

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		return NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
	}

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
	}

	// Auth check
	if ok := isUserAllowedToDelete(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to expunge this object")
	}

	// Check state
	if dbObject.IsExpunged {
		return NewAppError(http.StatusGone, err, "The referenced object no longer exists.")
	}

	dbObject.ModifiedBy = caller.DistinguishedName
	dbObject.ChangeToken = requestObject.ChangeToken
	err = dao.ExpungeObject(user, dbObject, true)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "DAO Error expunging object")
	}

	apiResponse := mapping.MapODObjectToExpungedObjectResponse(&dbObject).WithCallerPermission(protocolCaller(caller))

	objectID := hex.EncodeToString(requestObject.ID)

	gem.Action = "delete"
	gem.Payload = events.ObjectDriveEvent{
		ObjectID:     objectID,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: false,
		SessionID:    session,
	}
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)
	return nil
}

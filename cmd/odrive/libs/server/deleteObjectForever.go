package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) deleteObjectForever(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, ok := CallerFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	session := SessionIDFromContext(ctx)

	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		return NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
	}
	user.Snippets = snippetFields
	dao := DAOFromContext(ctx)

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		return NewAppError(400, err, "Error parsing JSON")
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	if ok := isUserAllowedToDelete(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to expunge this object")
	}

	if dbObject.IsExpunged {
		return NewAppError(410, err, "The referenced object no longer exists.")
	}

	dbObject.ModifiedBy = caller.DistinguishedName
	dbObject.ChangeToken = requestObject.ChangeToken
	err = dao.ExpungeObject(user, dbObject, true)
	if err != nil {
		return NewAppError(500, err, "DAO Error expunging object")
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

package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) deleteObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, errors.New("Could not get caller from context"), "Invalid caller.")
	}
	session := SessionIDFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
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

	// Get object
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		return NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
	}
	dbObject, err := dao.GetObject(requestObject, false)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
	}

	// Auth check
	if ok := isUserAllowedToDelete(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to delete this object")
	}

	// State check
	if dbObject.IsDeleted {
		// Deleted already
		switch {
		case dbObject.IsExpunged:
			return NewAppError(http.StatusGone, err, "The referenced object no longer exists.")
		default:
		}
	} else {
		// ok to change
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ChangeToken = requestObject.ChangeToken
		err = dao.DeleteObject(user, dbObject, true)
		if err != nil {
			return NewAppError(http.StatusInternalServerError, err, "DAO Error deleting object")
		}
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToDeletedObjectResponse(&dbObject).WithCallerPermission(protocolCaller(caller))

	gem.Action = "delete"
	gem.Payload = events.ObjectDriveEvent{
		ObjectID:     apiResponse.ID,
		ChangeToken:  requestObject.ChangeToken,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: false,
		SessionID:    session,
	}
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)
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

package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectFromTrash(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	caller, _ := CallerFromContext(ctx)
	session := SessionIDFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)

	// Get the change token off the request.
	changeToken, err := protocol.NewChangeTokenStructFromJSONBody(r.Body)
	if err != nil {
		return NewAppError(http.StatusBadRequest, err, "Unexpected change token")
	}

	requestObject, err := parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		code, err, msg := getObjectDAOError(err)
		return NewAppError(code, err, msg)
	}

	if dbObject.IsExpunged {
		return NewAppError(410, errors.New("Cannot undelete an expunged object"), "Object was expunged")
	}

	if dbObject.IsAncestorDeleted {
		return NewAppError(405, errors.New("Cannot undelete an object with a deleted parent"), "Object has deleted ancestor")
	}

	if dbObject.ChangeToken != changeToken.ChangeToken {
		err := errors.New("Changetoken in database does not match client changeToken")
		return NewAppError(400, err, "Invalid changeToken.")
	}

	if ok := isUserAllowedToDelete(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to undelete this object")
	}

	dbObject.ModifiedBy = caller.DistinguishedName

	unDeletedObj, err := dao.UndeleteObject(&dbObject)
	if err != nil {
		return NewAppError(500, err, "Error restoring object")
	}

	apiResponse := mapping.MapODObjectToObject(&unDeletedObj).WithCallerPermission(protocolCaller(caller))

	gem.Action = "undelete"
	gem.Payload = events.ObjectDriveEvent{
		ObjectID:     apiResponse.ID,
		ChangeToken:  apiResponse.ChangeToken,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: true,
		SessionID:    session,
	}
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)
	return nil
}

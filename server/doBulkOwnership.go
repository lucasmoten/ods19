package server

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) doBulkOwnership(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	gem, _ := GEMFromContext(ctx)
	session := SessionIDFromContext(ctx)
	captured, _ := CaptureGroupsFromContext(ctx)

	// Get object
	if r.Header.Get("Content-Type") != "application/json" {
		return NewAppError(http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
	}

	newOwner := captured["newOwner"]

	var objects []protocol.ObjectVersioned
	var bytes []byte
	limit := 5 * 1024 * 1024
	bytes, err := ioutil.ReadAll(io.LimitReader(r.Body, int64(limit)))

	if err != nil {
		return NewAppError(400, err, "Cannot unmarshal list of IDs", zap.String("baddata", string(bytes)))
	}
	err = json.Unmarshal(bytes, &objects)
	if err != nil {
		return NewAppError(400, err, "Cannot parse list of IDs", zap.String("baddata", string(bytes)))
	}

	bulkResponse := make([]protocol.ObjectError, 0)
	for _, o := range objects {
		changeRequest := protocol.ChangeOwnerRequest{
			ID:          o.ObjectID,
			ChangeToken: o.ChangeToken,
			NewOwner:    newOwner,
		}
		requestObject, err := mapping.MapChangeOwnerRequestToODObject(&changeRequest)

		if err != nil {
			return NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
		}
		requestObject.ChangeToken = o.ChangeToken
		dbObject, err := dao.GetObject(requestObject, true)
		if err != nil {
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ObjectID,
					Error:    err.Error(),
					Msg:      "Error retrieving object",
					Code:     400,
				},
			)
			continue
		}

		// Auth check
		okToUpdate, updatePermission := isUserAllowedToUpdateWithPermission(ctx, &dbObject)
		if !okToUpdate {
			return NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
		}
		if !aacAuth.IsUserOwner(caller.DistinguishedName, getKnownResourceStringsFromUserGroups(ctx), dbObject.OwnedBy.String) {
			return NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User must be an object owner to transfer ownership of the object")
		}

		var code int
		var msg string
		var errCause error

		_, herr := changeOwnerRaw(
			&requestObject, &dbObject,
			&updatePermission,
			aacAuth,
			caller,
			dao,
		)
		if herr != nil {
			errCause = herr.Error
			code = herr.Code
			msg = herr.Msg
		}

		if herr != nil {
			errMsg := errCause.Error()
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ObjectID,
					Error:    errMsg,
					Msg:      msg,
					Code:     code,
				},
			)
			continue
		}

		bulkResponse = append(
			bulkResponse,
			protocol.ObjectError{
				ObjectID: o.ObjectID,
				Error:    "",
				Msg:      "",
				Code:     200,
			},
		)

		apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller))

		gem.Action = "update"
		gem.Payload = events.ObjectDriveEvent{
			ObjectID:     apiResponse.ID,
			ChangeToken:  apiResponse.ChangeToken,
			UserDN:       caller.DistinguishedName,
			StreamUpdate: false,
			SessionID:    session,
		}
		h.EventQueue.Publish(gem)
	}
	jsonResponse(w, bulkResponse)
	return nil
}

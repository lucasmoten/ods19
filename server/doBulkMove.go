package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"encoding/hex"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) doBulkMove(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	gem, _ := GEMFromContext(ctx)
	session := SessionIDFromContext(ctx)

	var objects []protocol.MoveObjectRequest
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return NewAppError(400, err, "Cannot unmarshal list of IDs")
	}
	err = json.Unmarshal(bytes, &objects)
	if err != nil {
		return NewAppError(400, err, "Cannot parse list of IDs")
	}

	bulkResponse := make([]protocol.ObjectError, 0)
	for _, o := range objects {
		id, err := hex.DecodeString(o.ID)
		if err != nil {
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ID,
					Error:    err.Error(),
					Msg:      "Cannot decode object id",
					Code:     400,
				},
			)
			continue
		}

		requestObject := models.ODObject{
			ID:          id,
			ChangeToken: o.ChangeToken,
		}
		dbObject, err := dao.GetObject(requestObject, true)
		if err != nil {
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ID,
					Error:    err.Error(),
					Msg:      "Error retrieving object",
					Code:     400,
				},
			)
			continue
		}

		code, errCause, msg := moveObjectRaw(
			dao,
			ctx,
			caller,
			getKnownResourceStringsFromUserGroups(ctx),
			aacAuth,
			requestObject,
			&dbObject,
		)

		if errCause != nil {
			errMsg := errCause.Error()
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ID,
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
				ObjectID: o.ID,
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

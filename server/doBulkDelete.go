package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"encoding/hex"

	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) doBulkDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	user, _ := UserFromContext(ctx)
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, errors.New("Could not get caller from context"), "Invalid caller.")
	}
	session := SessionIDFromContext(ctx)
	gem, _ := GEMFromContext(ctx)

	var objects []protocol.ObjectVersioned
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return NewAppError(400, err, "Cannot unmarshal list of IDs")
	}
	json.Unmarshal(bytes, &objects)

	bulkResponse := make([]protocol.ObjectError, 0)
	for _, o := range objects {
		id, err := hex.DecodeString(o.ObjectID)
		if err != nil {
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ObjectID,
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
					ObjectID: o.ObjectID,
					Error:    err.Error(),
					Msg:      "Error retrieving object",
					Code:     400,
				},
			)
			continue
		}

		err = dao.DeleteObject(user, dbObject, true)
		if err != nil {
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ObjectID,
					Error:    err.Error(),
					Msg:      "Cannot decode object id",
					Code:     400,
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
	}
	jsonResponse(w, bulkResponse)
	return nil
}

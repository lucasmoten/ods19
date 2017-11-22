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
	"decipher.com/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) doBulkMove(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MOVE")

	var objects []protocol.MoveObjectRequest
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Cannot read list of IDs")
		h.publishError(gem, herr)
		return herr
	}
	err = json.Unmarshal(bytes, &objects)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Cannot parse list of IDs")
		h.publishError(gem, herr)
		return herr
	}

	var bulkResponse []protocol.ObjectError
	w.Header().Set("Status", (string)(http.StatusOK))
	for _, o := range objects {
		gem = ResetBulkItem(gem)
		id, err := hex.DecodeString(o.ID)
		if err != nil {
			herr := NewAppError(http.StatusBadRequest, err, "Cannot decode object id")
			h.publishError(gem, herr)
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}

		requestObject := models.ODObject{
			ID:          id,
			ChangeToken: o.ChangeToken,
		}
		gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
		gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))
		dbObject, err := dao.GetObject(requestObject, true)
		if err != nil {
			herr := NewAppError(http.StatusBadRequest, err, "Error retrieving object")
			h.publishError(gem, herr)
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}
		auditOriginal := NewResourceFromObject(dbObject)

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
			herr := NewAppError(code, errCause, msg)
			h.publishError(gem, herr)
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}

		auditModified := NewResourceFromObject(dbObject)

		bulkResponse = append(
			bulkResponse,
			protocol.ObjectError{
				ObjectID: o.ID,
				Error:    "",
				Msg:      "",
				Code:     http.StatusOK,
			},
		)

		apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller))

		gem.Payload.ChangeToken = apiResponse.ChangeToken
		gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
		gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
		h.publishSuccess(gem, w)

	}
	jsonResponse(w, bulkResponse)
	return nil
}

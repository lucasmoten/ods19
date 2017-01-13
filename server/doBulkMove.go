package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"encoding/hex"

	"decipher.com/object-drive-server/auth"
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
		herr := NewAppError(400, err, "Cannot read list of IDs")
		h.publishError(gem, herr)
		return herr
	}
	err = json.Unmarshal(bytes, &objects)
	if err != nil {
		herr := NewAppError(400, err, "Cannot parse list of IDs")
		h.publishError(gem, herr)
		return herr
	}

	var bulkResponse []protocol.ObjectError
	for _, o := range objects {
		gem.ID = newGUID()
		gem.Payload.Audit = audit.WithID(gem.Payload.Audit, "guid", gem.ID)
		gem.Payload.Audit.Resources = nil
		gem.Payload.Audit.ModifiedPairList = nil
		id, err := hex.DecodeString(o.ID)
		if err != nil {
			herr := NewAppError(400, err, "Cannot decode object id")
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
			herr := NewAppError(400, err, "Error retrieving object")
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
				Code:     200,
			},
		)

		apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller))

		gem.Payload.ChangeToken = apiResponse.ChangeToken
		gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
		h.publishSuccess(gem, r)

	}
	jsonResponse(w, bulkResponse)
	return nil
}

package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"encoding/hex"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) doBulkDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	user, _ := UserFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "delete"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventDelete")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "REMOVE")

	var objects []protocol.ObjectVersioned
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
	w.Header().Set("Status","200")
	for _, o := range objects {
		gem = ResetBulkItem(gem)
		id, err := hex.DecodeString(o.ObjectID)
		if err != nil {
			herr := NewAppError(400, err, "Cannot decode object id")
			h.publishError(gem, herr)
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ObjectID,
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
					ObjectID: o.ObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}
		gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(dbObject))
		gem.Payload.ChangeToken = dbObject.ChangeToken
		err = dao.DeleteObject(user, dbObject, true)
		if err != nil {
			herr := NewAppError(400, err, "DAO Error deleting object")
			h.publishError(gem, herr)
			bulkResponse = append(bulkResponse,
				protocol.ObjectError{
					ObjectID: o.ObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
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

		// reget the object so that changetoken and deleteddate are correct
		dbObject, err = dao.GetObject(requestObject, false)
		gem.Payload.ChangeToken = dbObject.ChangeToken

		h.publishSuccess(gem, w)

	}
	jsonResponse(w, bulkResponse)
	return nil
}

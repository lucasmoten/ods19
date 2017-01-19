package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) doBulkOwnership(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "OWNERSHIP_MODIFY")
	captured, _ := CaptureGroupsFromContext(ctx)

	// Get object
	if r.Header.Get("Content-Type") != "application/json" {
		herr := NewAppError(http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		h.publishError(gem, herr)
		return herr
	}

	newOwner := captured["newOwner"]

	var objects []protocol.ObjectVersioned
	var bytes []byte
	limit := 5 * 1024 * 1024
	bytes, err := ioutil.ReadAll(io.LimitReader(r.Body, int64(limit)))

	if err != nil {
		herr := NewAppError(400, err, "Cannot unmarshal list of IDs", zap.String("baddata", string(bytes)))
		h.publishError(gem, herr)
		return herr
	}
	err = json.Unmarshal(bytes, &objects)
	if err != nil {
		herr := NewAppError(400, err, "Cannot parse list of IDs", zap.String("baddata", string(bytes)))
		h.publishError(gem, herr)
		return herr
	}

	var bulkResponse []protocol.ObjectError
	w.Header().Set("Status","200")
	for _, o := range objects {
		gem = ResetBulkItem(gem)

		changeRequest := protocol.ChangeOwnerRequest{
			ID:          o.ObjectID,
			ChangeToken: o.ChangeToken,
			NewOwner:    newOwner,
		}
		requestObject, err := mapping.MapChangeOwnerRequestToODObject(&changeRequest)

		if err != nil {
			herr := NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
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
		gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
		gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

		requestObject.ChangeToken = o.ChangeToken
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
		auditOriginal := NewResourceFromObject(dbObject)

		// Auth check
		okToUpdate, updatePermission := isUserAllowedToUpdateWithPermission(ctx, &dbObject)
		if !okToUpdate {
			herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
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
		if !aacAuth.IsUserOwner(caller.DistinguishedName, getKnownResourceStringsFromUserGroups(ctx), dbObject.OwnedBy.String) {
			herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User must be an object owner to transfer ownership of the object")
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
			errMsg := errCause.Error()
			h.publishError(gem, herr)
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
		auditModified := NewResourceFromObject(dbObject)

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

		gem.Payload.ChangeToken = apiResponse.ChangeToken
		gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
		h.publishSuccess(gem, w)
	}
	jsonResponse(w, bulkResponse)
	return nil
}

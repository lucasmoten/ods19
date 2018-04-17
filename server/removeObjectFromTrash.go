package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/deciphernow/object-drive-server/events"
	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/services/audit"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectFromTrash(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "undelete"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MODIFY")

	// Get the change token off the request.
	changeToken, err := protocol.NewChangeTokenStructFromJSONBody(r.Body)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Unexpected change token")
		h.publishError(gem, herr)
		return herr
	}

	requestObject, err := parseGetObjectRequest(ctx)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing URI")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		code, msg, err := getObjectDAOError(err)
		herr := NewAppError(code, err, msg)
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)

	if dbObject.IsExpunged {
		herr := NewAppError(http.StatusGone, errors.New("Cannot undelete an expunged object"), "Object was expunged")
		h.publishError(gem, herr)
		return herr
	}

	if dbObject.IsAncestorDeleted {
		herr := NewAppError(http.StatusConflict, errors.New("Cannot undelete an object with a deleted parent"), "Object has deleted ancestor")
		h.publishError(gem, herr)
		return herr
	}

	if dbObject.ChangeToken != changeToken.ChangeToken {
		err := errors.New("Changetoken in database does not match client changeToken")
		herr := NewAppError(http.StatusConflict, err, "Invalid changeToken.")
		h.publishError(gem, herr)
		return herr
	}

	if ok := isUserAllowedToDelete(ctx, &dbObject); !ok {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to undelete this object")
		h.publishError(gem, herr)
		return herr
	}

	dbObject.ModifiedBy = caller.DistinguishedName

	unDeletedObj, err := dao.UndeleteObject(&dbObject)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error restoring object")
		h.publishError(gem, herr)
		return herr
	}

	apiResponse := mapping.MapODObjectToObject(&unDeletedObj).WithCallerPermission(protocolCaller(caller))
	auditModified := NewResourceFromObject(unDeletedObj)

	gem.Payload.ChangeToken = unDeletedObj.ChangeToken
	gem.Payload.StreamUpdate = unDeletedObj.ContentSize.Int64 > 0
	gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
	gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

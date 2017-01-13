package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/audit"

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
		herr := NewAppError(500, err, "Error parsing URI")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithResource(gem.Payload.Audit, "", "", 0, "", "", hex.EncodeToString(requestObject.ID))

	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		code, msg, err := getObjectDAOError(err)
		herr := NewAppError(code, err, msg)
		h.publishError(gem, herr)
		return herr
	}

	if dbObject.IsExpunged {
		herr := NewAppError(410, errors.New("Cannot undelete an expunged object"), "Object was expunged")
		h.publishError(gem, herr)
		return herr
	}

	if dbObject.IsAncestorDeleted {
		herr := NewAppError(405, errors.New("Cannot undelete an object with a deleted parent"), "Object has deleted ancestor")
		h.publishError(gem, herr)
		return herr
	}

	if dbObject.ChangeToken != changeToken.ChangeToken {
		err := errors.New("Changetoken in database does not match client changeToken")
		herr := NewAppError(400, err, "Invalid changeToken.")
		h.publishError(gem, herr)
		return herr
	}

	if ok := isUserAllowedToDelete(ctx, &dbObject); !ok {
		herr := NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to undelete this object")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.Audit.Resources = gem.Payload.Audit.Resources[:len(gem.Payload.Audit.Resources)-1]
	gem.Payload.Audit = audit.WithResource(gem.Payload.Audit, dbObject.Name, "", dbObject.ContentSize.Int64, "OBJECT", dbObject.TypeName.String, hex.EncodeToString(requestObject.ID))
	gem.Payload.ChangeToken = dbObject.ChangeToken

	dbObject.ModifiedBy = caller.DistinguishedName

	unDeletedObj, err := dao.UndeleteObject(&dbObject)
	if err != nil {
		herr := NewAppError(500, err, "Error restoring object")
		h.publishError(gem, herr)
		return herr
	}

	apiResponse := mapping.MapODObjectToObject(&unDeletedObj).WithCallerPermission(protocolCaller(caller))

	gem.Payload.ChangeToken = unDeletedObj.ChangeToken
	gem.Payload.StreamUpdate = true
	gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(
		*gem.Payload.Audit.Resources[0],
		*gem.Payload.Audit.Resources[0],
	))
	h.publishSuccess(gem, r)

	jsonResponse(w, apiResponse)
	return nil
}

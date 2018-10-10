package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"encoding/hex"

	"bitbucket.di2e.net/dime/object-drive-server/events"
	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) doBulkDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	user, _ := UserFromContext(ctx)
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, errors.New("Could not get caller from context"), "Invalid caller.")
	}

	gem, _ := GEMFromContext(ctx)
	gem.Action = "delete"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventDelete")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "REMOVE")

	var objects []protocol.ObjectVersioned
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
	// Limit bulk delete operation to 1000 items
	if len(objects) > 1000 {
		herr := NewAppError(http.StatusBadRequest, err, "Cannot delete more then 1000 objects at a time")
		h.publishError(gem, herr)
		return herr
	}

	var bulkResponse []protocol.ObjectError
	for _, o := range objects {
		gem = ResetBulkItem(gem)
		id, err := hex.DecodeString(o.ObjectID)
		if err != nil {
			herr := NewAppError(http.StatusBadRequest, err, "Cannot decode object id")
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
			herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
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

		// Auth check
		if ok := isUserAllowedToDelete(ctx, &dbObject); !ok {
			herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to delete this object")
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

		// State check
		if dbObject.IsDeleted {
			// Deleted already
			switch {
			case dbObject.IsExpunged:
				herr := NewAppError(http.StatusGone, err, "The referenced object no longer exists.")
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
			default:
				// Just ignore files already deleted.
			}
		} else {
			// ok to change
			dbObject.ModifiedBy = caller.DistinguishedName
			dbObject.ChangeToken = requestObject.ChangeToken
			err = dao.DeleteObject(user, dbObject, true)
			if err != nil {
				herr := NewAppError(http.StatusInternalServerError, err, "DAO Error deleting object")
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
		}

		bulkResponse = append(
			bulkResponse,
			protocol.ObjectError{
				ObjectID: o.ObjectID,
				Error:    "",
				Msg:      "",
				Code:     http.StatusOK,
			},
		)

		// reget the object so that changetoken and deleteddate are correct
		dbObject, err = dao.GetObject(requestObject, false)
		gem.Payload.ChangeToken = dbObject.ChangeToken
		gem.Payload = events.WithEnrichedPayload(gem.Payload, mapping.MapODObjectToObject(&dbObject))
		h.publishSuccess(gem, w)

	}
	jsonResponse(w, bulkResponse)
	return nil
}

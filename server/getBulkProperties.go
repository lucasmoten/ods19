package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"encoding/hex"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"golang.org/x/net/context"
)

func (h AppServer) getBulkProperties(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	logger := LoggerFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	var objects protocol.ObjectIds
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Cannot unmarshal list of IDs")
		h.publishError(gem, herr)
		return herr
	}
	json.Unmarshal(bytes, &objects)

	// Check size of request to ensure within bounds for max page size
	if len(objects.ObjectIds) > int(h.Conf.MaxPageSize) {
		herr := NewAppError(http.StatusBadRequest, err, fmt.Sprintf("too many objects requested. limited to %d", h.Conf.MaxPageSize))
		h.publishError(gem, herr)
		return herr
	}

	var bulkResponse protocol.ObjectResultset
	bulkResponse.PageNumber = 1
	bulkResponse.PageCount = 1
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	w.Header().Set("Status", "200")
	for _, requestObjectID := range objects.ObjectIds {
		gem = ResetBulkItem(gem)
		id, err := hex.DecodeString(requestObjectID)
		if err != nil {
			herr := NewAppError(http.StatusBadRequest, err, "Cannot decode object id")
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}
		requestObject := models.ODObject{
			ID: id,
		}
		gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
		gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

		//NOTE: we do not want to do this all in one transaction, because we are doing long ops to check each object with AAC.
		//  Just do them in independent transactions in order to not tie up the database with long running transactions.
		//  We could re-order to fetch all and purge things that won't pass AAC checks later.
		// Retrieve existing object from the data store
		dbObject, err := dao.GetObject(requestObject, true)
		if err != nil {
			code, msg, err := getObjectDAOError(err)
			herr := NewAppError(code, err, msg)
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}
		gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(dbObject))
		gem.Payload.ChangeToken = dbObject.ChangeToken

		// Check if the user has permissions to read the ODObject
		//		Permission.grantee matches caller, and AllowRead is true
		if ok := isUserAllowedToRead(ctx, &dbObject); !ok {
			msg := "Forbidden - User does not have permission to read/view this object"
			herr := NewAppError(http.StatusForbidden, fmt.Errorf(msg), msg)
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}

		if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
			msg := "Forbidden - User does not have permission to read/view this object"
			herr := NewAppError(http.StatusForbidden, err, msg)
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}

		if ok, code, err := isExpungedOrAnscestorDeletedErr(dbObject); !ok {
			msg := "Gone - Requested object or its ancestor has been deleted, expunged, or otherwise no longer exists"
			herr := NewAppError(code, err, msg)
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
		}

		if dbObject.IsDeleted {
			msg := "Gone - Requested object or its ancestor has been deleted, expunged, or otherwise no longer exists"
			herr := NewAppError(http.StatusGone, fmt.Errorf(msg), msg)
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}

		parents, err := dao.GetParents(dbObject)
		if err != nil {
			herr := NewAppError(http.StatusInternalServerError, err, "error retrieving object parents")
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}

		filtered := redactParents(ctx, aacAuth, parents)
		if herr := errOnDeletedParents(parents); herr != nil {
			h.publishError(gem, herr)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    herr.Error.Error(),
					Msg:      herr.Msg,
					Code:     herr.Code,
				},
			)
			continue
		}

		crumbs := breadcrumbsFromParents(filtered)
		apiResponse := mapping.MapODObjectToObject(&dbObject).
			WithCallerPermission(protocolCaller(caller)).
			WithBreadcrumbs(crumbs)

		bulkResponse.Objects = append(bulkResponse.Objects, apiResponse)
		bulkResponse.TotalRows++
		bulkResponse.PageSize++
		h.publishSuccess(gem, w)
	}
	jsonResponse(w, bulkResponse)
	return nil
}

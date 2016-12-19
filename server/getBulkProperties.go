package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"encoding/hex"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) getBulkProperties(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)

	var objects protocol.ObjectIds
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return NewAppError(400, err, "Cannot unmarshal list of IDs")
	}
	json.Unmarshal(bytes, &objects)

	var bulkResponse protocol.ObjectResultset
	bulkResponse.PageNumber = 1
	bulkResponse.PageCount = 1
	for _, requestObjectID := range objects.ObjectIds {
		id, err := hex.DecodeString(requestObjectID)
		if err != nil {
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Error:    err.Error(),
					Msg:      "Cannot decode object id",
					Code:     400,
				},
			)
			continue
		}
		requestObject := models.ODObject{
			ID: id,
		}
		//NOTE: we do not want to do this all in one transaction, because we are doing long ops to check each object with AAC.
		//  Just do them in independent transactions in order to not tie up the database with long running transactions.
		//  We could re-order to fetch all and purge things that won't pass AAC checks later.
		// Retrieve existing object from the data store
		dbObject, err := dao.GetObject(requestObject, true)
		if err != nil {
			code, msg, err := getObjectDAOError(err)
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Msg:      msg,
					Error:    err.Error(),
					Code:     code,
				},
			)
			continue
		}

		// Check if the user has permissions to read the ODObject
		//		Permission.grantee matches caller, and AllowRead is true
		if ok := isUserAllowedToRead(ctx, &dbObject); !ok {
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Msg:      "Forbidden - User does not have permission to read/view this object",
					Error:    "Forbidden - User does not have permission to read/view this object",
					Code:     403,
				},
			)
			continue
		}
		aacAuth := auth.NewAACAuth(logger, h.AAC)
		if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Msg:      "Forbidden - User does not have permission to read/view this object",
					Error:    err.Error(),
					Code:     403,
				},
			)
			continue
		}

		if ok, _, err := isExpungedOrAnscestorDeletedErr(dbObject); !ok {
			msg := "Gone - Requested object or its ancestor has been deleted, expunged, or otherwise no longer exists"
			errStr := msg
			if err != nil {
				errStr = err.Error()
			}
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Msg:      msg,
					Error:    errStr,
					Code:     410,
				},
			)
			continue
		}

		if dbObject.IsDeleted {
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Msg:      "Gone - Requested object or its ancestor has been deleted, expunged, or otherwise no longer exists",
					Error:    "Gone - Requested object has been deleted, expunged, or otherwise no longer exists",
					Code:     410,
				},
			)
			continue
		}

		parents, err := dao.GetParents(dbObject)
		if err != nil {
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Msg:      "error retrieving object parents",
					Error:    err.Error(),
					Code:     500,
				},
			)
			continue
		}

		filtered := redactParents(ctx, aacAuth, parents)
		if err := errOnDeletedParents(parents); err != nil {
			bulkResponse.ObjectErrors = append(bulkResponse.ObjectErrors,
				protocol.ObjectError{
					ObjectID: requestObjectID,
					Msg:      "deleted parents",
					Error:    fmt.Sprintf("%v", err),
					Code:     500,
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
	}
	jsonResponse(w, bulkResponse)
	return nil
}

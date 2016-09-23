package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) getObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)

	requestObject, err := parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		code, err, msg := getObjectDAOError(err)
		return NewAppError(code, err, msg)
	}

	// Check if the user has permissions to read the ODObject
	//		Permission.grantee matches caller, and AllowRead is true
	if ok := isUserAllowedToRead(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to read/view this object")
	}

	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
	if err != nil {
		// TODO: Isolate different error types
		//return NewAppError(502, err, "Error communicating with authorization service")
		return NewAppError(403, err, err.Error())
	}
	if !hasAACAccess {
		return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for object ACM")
	}

	if ok, code, err := isExpungedOrAnscestorDeletedErr(dbObject); !ok {
		return NewAppError(code, err, "expunged or ancesor deleted")
	}

	if dbObject.IsDeleted {
		apiResponse := mapping.MapODObjectToDeletedObject(&dbObject).WithCallerPermission(protocolCaller(caller))
		jsonResponse(w, apiResponse)
		return nil
	}

	parents, err := dao.GetParents(dbObject)
	if err != nil {
		return NewAppError(500, err, "error retrieving object parents")
	}

	parents = redactParents(ctx, h, parents)
	if err := errOnDeletedParents(parents); err != nil {
		return err
	}
	crumbs := breadcrumbsFromParents(parents)

	apiResponse := mapping.MapODObjectToObject(&dbObject).
		WithCallerPermission(protocolCaller(caller)).
		WithBreadcrumbs(crumbs)
	jsonResponse(w, apiResponse)
	return nil
}

func isExpungedOrAnscestorDeletedErr(obj models.ODObject) (ok bool, code int, err error) {
	switch {
	case obj.IsExpunged:
		return false, 410, errors.New("The object no longer exists.")
	case obj.IsAncestorDeleted:
		return false, 405, errors.New("The object cannot be retreived because an ancestor is deleted.")
	}
	// NOTE the obj.IsDeleted case is not an error for getObject. Getting metadata
	// about a trashed object with IsDeleted = true is still okay.
	return true, 0, nil
}

func getObjectDAOError(err error) (int, error, string) {
	if err != nil {
		//We can't use equality checks on the error
		if err.Error() == dao.ErrNoRows.Error() {
			return 404, dao.ErrNoRows, "Not found"
		}
	}
	switch err {
	case dao.ErrMissingID:
		return 400, err, "Must provide ID field"
	default:
		return 500, err, "Error retrieving object"
	}
}

func redactParents(ctx context.Context, h AppServer, parents []models.ODObject) []models.ODObject {
	logger := LoggerFromContext(ctx)
	// for each parent, check authorization to read and AAC call
	for i, parent := range parents {
		if ok := isUserAllowedToRead(ctx, h.MasterKey, &parent); !ok {
			parents[i].Name = "Not Authorized"
		}
		if _, err := h.isUserAllowedForObjectACM(ctx, &parent); err != nil {
			parents[i].Name = "Not Authorized"
			if !IsDeniedAccess(err) {
				// already redacted, log possible 502 and continue
				logger.Error("AAC error checking parent", zap.Object("err", err))
			}
		}
	}
	return parents
}

func errOnDeletedParents(parents []models.ODObject) *AppError {
	for _, parent := range parents {
		if parent.IsDeleted {
			return NewAppError(500, errors.New("cannot get properties on object with deleted ancestor"), "")
		}
	}
	return nil
}

func breadcrumbsFromParents(parents []models.ODObject) []protocol.Breadcrumb {
	var crumbs []protocol.Breadcrumb

	for _, p := range parents {
		b := protocol.Breadcrumb{
			ID:       hex.EncodeToString(p.ID),
			ParentID: hex.EncodeToString(p.ParentID),
			Name:     p.Name,
		}
		crumbs = append(crumbs, b)
	}
	return crumbs
}

func parseGetObjectRequest(ctx context.Context) (models.ODObject, error) {
	var requestObject models.ODObject

	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return requestObject, errors.New("Could not get capture groups")
	}

	if captured["objectId"] == "" {
		return requestObject, errors.New("Could not extract objectId from URI")
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return requestObject, errors.New("Invalid objectid in URI.")
	}
	requestObject.ID = bytesObjectID
	return requestObject, nil
}

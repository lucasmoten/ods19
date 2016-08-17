package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) getObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)

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
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for object ACM")
	}

	if ok, code, err := isExpungedOrAnscestorDeletedErr(dbObject); !ok {
		return NewAppError(code, err, "expunged or ancesor deleted")
	}

	if dbObject.IsDeleted {
		jsonResponse(w, mapping.MapODObjectToDeletedObject(&dbObject))
	} else {
		jsonResponse(w, mapping.MapODObjectToObject(&dbObject))
	}

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

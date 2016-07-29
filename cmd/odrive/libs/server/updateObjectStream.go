package server

import (
	"encoding/json"
	"errors"
	"net/http"

	db "decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"github.com/uber-go/zap"
	"golang.org/x/net/context"
)

// updateObjectStream ...
func (h AppServer) updateObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	var drainFunc func()

	logger := LoggerFromContext(ctx)

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	var requestObjectWithIDFromURI models.ODObject
	var err error

	requestObjectWithIDFromURI, err = parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObjectWithIDFromURI, true)
	if err != nil {
		if err.Error() == db.ErrNoRows.Error() {
			return NewAppError(404, err, "Not found")
		}
		return NewAppError(500, err, "Error retrieving object")
	}

	if len(dbObject.ID) == 0 {
		return NewAppError(400, err, "Object for update doesn't have an id")
	}

	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted:
			return NewAppError(405, err, "The object cannot be modified because an ancestor is deleted.")
		default:
			return NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
		}
	}

	//We need a name for the new text, and a new iv
	dbObject.ContentConnector.String = utils.CreateRandomName()
	dbObject.EncryptIV = utils.CreateIV()
	// Check if the user has permissions to update the ODObject
	var grant models.ODObjectPermission
	if ok, grant = isUserAllowedToUpdateWithPermission(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	hasAACAccessToOLDACM, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccessToOLDACM {
		return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for existing object ACM")
	}

	//Do an upload that is basically the same as for a new object.
	multipartReader, err := r.MultipartReader()
	if err != nil {
		return NewAppError(400, err, "unable to open multipart reader")
	}
	drainFunc, herr, err := h.acceptObjectUpload(ctx, multipartReader, &dbObject, &grant, false)
	if herr != nil {
		return herr
	}

	// TODO: This seems weird. why was everything previously manipulating the dbObject (call to h.acceptObjectUpload) ?
	// Assign existing permissions from the database object to the request object
	requestObjectWithIDFromURI.Permissions = dbObject.Permissions
	requestObjectWithIDFromURI.RawAcm.Valid = dbObject.RawAcm.Valid
	requestObjectWithIDFromURI.RawAcm.String = dbObject.RawAcm.String
	logger.Info("acm", zap.String("requestObjectWithIDFromURI.RawAcm", requestObjectWithIDFromURI.RawAcm.String))
	hasAACAccess := false

	err = h.flattenACM(logger, &requestObjectWithIDFromURI)
	if err != nil {
		return NewAppError(400, err, "ACM provided could not be flattened")
	}
	if herr := normalizeObjectReadPermissions(ctx, &requestObjectWithIDFromURI); herr != nil {
		return herr
	}
	// Final access check against altered ACM
	hasAACAccess, err = h.isUserAllowedForObjectACM(ctx, &requestObjectWithIDFromURI)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for updated object ACM")
	}
	// copy grant.EncryptKey to all existing permissions:
	for idx, permission := range requestObjectWithIDFromURI.Permissions {
		models.CopyEncryptKey(h.MasterKey, &grant, &permission)
		models.CopyEncryptKey(h.MasterKey, &grant, &requestObjectWithIDFromURI.Permissions[idx])
	}
	// Assign ACM and permissions on request to dbObject
	dbObject.Permissions = requestObjectWithIDFromURI.Permissions
	dbObject.RawAcm.String = requestObjectWithIDFromURI.RawAcm.String

	dbObject.ModifiedBy = caller.DistinguishedName
	err = dao.UpdateObject(&dbObject)
	if err != nil {
		return NewAppError(500, err, "error storing object")
	}
	// Only start to upload into S3 after we have a database record
	go drainFunc()

	w.Header().Set("Content-Type", "application/json")
	link := mapping.MapODObjectToObject(&dbObject)
	data, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		return NewAppError(500, err, "could not unmarshal json data")
	}
	w.Write(data)

	return nil
}

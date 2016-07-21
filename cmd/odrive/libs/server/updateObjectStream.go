package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"github.com/uber-go/zap"
	"golang.org/x/net/context"
)

/**
Almost all code is similar to that of createObject.go, so reuse much code from there.
*/
func (h AppServer) updateObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	var drainFunc func()

	logger := LoggerFromContext(ctx)

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	var requestObject models.ODObject
	var err error

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
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
	requestObject.Permissions = dbObject.Permissions
	requestObject.RawAcm.Valid = dbObject.RawAcm.Valid
	requestObject.RawAcm.String = dbObject.RawAcm.String
	logger.Info("acm", zap.String("requestObject.RawAcm", requestObject.RawAcm.String))
	hasAACAccess := false
	// Flatten ACM, then Normalize Read Permissions against ACM f_share
	// hasAACAccess, err := h.flattenACMAndCheckAccess(ctx, &requestObject)
	// if err != nil {
	// 	return NewAppError(400, err, "ACM provided could not be flattened")
	// }
	// if !hasAACAccess {
	// 	return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for updated object ACM")
	// }
	err = h.flattenACM(logger, &requestObject)
	if err != nil {
		return NewAppError(400, err, "ACM provided could not be flattened")
	}
	if herr := normalizeObjectReadPermissions(ctx, &requestObject); herr != nil {
		return herr
	}
	// Final access check against altered ACM
	hasAACAccess, err = h.isUserAllowedForObjectACM(ctx, &requestObject)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for updated object ACM")
	}
	// copy grant.EncryptKey to all existing permissions:
	for idx, permission := range requestObject.Permissions {
		models.CopyEncryptKey(h.MasterKey, &grant, &permission)
		models.CopyEncryptKey(h.MasterKey, &grant, &requestObject.Permissions[idx])
	}
	// Assign ACM and permissions on request to dbObject
	dbObject.Permissions = requestObject.Permissions
	dbObject.RawAcm.String = requestObject.RawAcm.String

	dbObject.ModifiedBy = caller.DistinguishedName
	err = dao.UpdateObject(&dbObject)
	if err != nil {
		//Note that if the DAO is not going to decide on a specific error code,
		// we *always* need to know if the error is due to bad user input,
		// a possible problem not under user control, and something that signifies a bug on our part.
		//
		// If we don't just return AppError, then we at least need to pass back a boolean or a constant
		// that classifies the error appropriately.  Otherwise, we need to return errors with more structure
		// than we have generically.
		//
		//4xx http codes are *good* because they caught bad input; possibly malicious.
		//5xx http codes signifies something *bad* that we must fix.
		//XXX get this back to returning a proper code
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

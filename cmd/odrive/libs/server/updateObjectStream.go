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

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	var grant *models.ODObjectPermission
	var requestObject models.ODObject
	var err error

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}

	// Retrieve existing object from the data store
	object, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	if len(object.ID) == 0 {
		return NewAppError(400, err, "Object for update doesn't have an id")
	}

	if object.IsDeleted {
		switch {
		case object.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case object.IsAncestorDeleted:
			return NewAppError(405, err, "The object cannot be modified because an ancestor is deleted.")
		default:
			return NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
		}
	}

	//We need a name for the new text, and a new iv
	object.ContentConnector.String = utils.CreateRandomName()
	object.EncryptIV = utils.CreateIV()
	// Check for update permission and capture a grant in the process
	for _, permission := range object.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowUpdate {
			grant = &permission
			break
		}
	}
	// Do we have permission ?
	if grant == nil {
		return NewAppError(403, nil, "Unauthorized")
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	hasAACAccessToOLDACM, err := h.isUserAllowedForObjectACM(ctx, &object)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccessToOLDACM {
		return NewAppError(403, err, "Unauthorized", zap.String("origination", "No access to old ACM on Update"), zap.String("acm", object.RawAcm.String))
	}

	//Descramble key (and rescramble when we go to save object back)
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)
	//Do an upload that is basically the same as for a new object.
	multipartReader, err := r.MultipartReader()
	if err != nil {
		return NewAppError(400, err, "unable to open multipart reader")
	}
	drainFunc, herr, err := h.acceptObjectUpload(ctx, multipartReader, &object, grant, false)
	if herr != nil {
		return herr
	}
	//Rescramble key
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)

	object.ModifiedBy = caller.DistinguishedName
	err = dao.UpdateObject(&object)
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
	link := mapping.MapODObjectToObject(&object)
	data, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		return NewAppError(500, err, "could not unmarshal json data")
	}
	w.Write(data)

	return nil
}

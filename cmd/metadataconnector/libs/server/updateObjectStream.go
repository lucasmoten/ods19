package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/cmd/metadataconnector/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"golang.org/x/net/context"
)

/**
Almost all code is similar to that of createObject.go, so reuse much code from there.
*/
func (h AppServer) updateObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var drainFunc func()

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	var grant *models.ODObjectPermission
	var requestObject models.ODObject
	var err error

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error parsing URI")
		return
	}

	// Retrieve existing object from the data store
	object, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error retrieving object")
		return
	}

	if len(object.ID) == 0 {
		sendErrorResponse(&w, 400, err, "Object for update doesn't have an id")
		return
	}

	if object.IsDeleted {
		switch {
		case object.IsExpunged:
			sendErrorResponse(&w, 410, err, "The object no longer exists.")
			return
		case object.IsAncestorDeleted:
			sendErrorResponse(&w, 405, err, "The object cannot be modified because an ancestor is deleted.")
			return
		default:
			sendErrorResponse(&w, 405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
			return
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
		sendErrorResponse(&w, 403, nil, "Unauthorized")
		return
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	hasAACAccessToOLDACM, err := h.isUserAllowedForObjectACM(ctx, &object)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error communicating with authorization service")
		return
	}
	if !hasAACAccessToOLDACM {
		sendErrorResponse(&w, 403, err, "Unauthorized")
		return
	}

	//Descramble key (and rescramble when we go to save object back)
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)
	//Do an upload that is basically the same as for a new object.
	multipartReader, err := r.MultipartReader()
	if err != nil {
		sendErrorResponse(&w, 400, err, "unable to open multipart reader")
		return
	}
	drainFunc, herr, err := h.acceptObjectUpload(ctx, multipartReader, &object, grant, false)
	if herr != nil {
		sendAppErrorResponse(&w, herr)
		return
	}
	//Rescramble key
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)

	object.ModifiedBy = caller.DistinguishedName
	err = h.DAO.UpdateObject(&object)
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
		sendError(&w, err, "error storing object")
		return
	}
	// Only start to upload into S3 after we have a database record
	go drainFunc()

	w.Header().Set("Content-Type", "application/json")
	link := mapping.MapODObjectToObject(&object)
	data, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		sendErrorResponse(&w, 500, err, "could not unmarshal json data")
		return
	}
	w.Write(data)

	countOKResponse()
}

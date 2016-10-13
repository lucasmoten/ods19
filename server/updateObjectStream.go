package server

import (
	"errors"
	"net/http"

	db "decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/utils"
	"github.com/uber-go/zap"
	"golang.org/x/net/context"
)

// updateObjectStream ...
func (h AppServer) updateObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	var drainFunc func()

	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	session := SessionIDFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	dp := h.DrainProvider

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
	var ok bool
	if ok, grant = isUserAllowedToUpdateWithPermission(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	if err = h.isUserAllowedForObjectACM(ctx, &dbObject); err != nil {
		return ClassifyObjectACMError(err)
	}

	multipartReader, err := r.MultipartReader()
	if err != nil {
		return NewAppError(400, err, "unable to open multipart reader")
	}
	drainFunc, herr, err := h.acceptObjectUpload(ctx, multipartReader, &dbObject, &grant, false)
	if herr != nil {
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}

	// TODO: This seems weird. why was everything previously manipulating the dbObject (call to h.acceptObjectUpload) ?
	// Assign existing permissions from the database object to the request object
	requestObjectWithIDFromURI.Permissions = dbObject.Permissions
	requestObjectWithIDFromURI.OwnedBy = models.ToNullString(dbObject.OwnedBy.String)
	requestObjectWithIDFromURI.RawAcm = models.ToNullString(dbObject.RawAcm.String)
	logger.Info("acm", zap.String("requestObjectWithIDFromURI.RawAcm", requestObjectWithIDFromURI.RawAcm.String))

	if err = h.flattenACM(logger, &requestObjectWithIDFromURI); err != nil {
		herr = ClassifyFlattenError(err)
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	if herr := normalizeObjectReadPermissions(ctx, &requestObjectWithIDFromURI); herr != nil {
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	// Final access check against altered ACM
	if err = h.isUserAllowedForObjectACM(ctx, &requestObjectWithIDFromURI); err != nil {
		herr = ClassifyObjectACMError(err)
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	consolidateChangingPermissions(&requestObjectWithIDFromURI)
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
		herr = NewAppError(500, err, "error storing object")
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	// Only start to upload into S3 after we have a database record
	go drainFunc()

	apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller))

	gem.Action = "update"
	gem.Payload = events.ObjectDriveEvent{
		ObjectID:     apiResponse.ID,
		ChangeToken:  apiResponse.ChangeToken,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: true,
		SessionID:    session,
	}
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)

	return nil
}

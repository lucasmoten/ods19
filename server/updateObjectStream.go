package server

import (
	"errors"
	"net/http"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/crypto"
	db "decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
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

	var requestObject models.ODObject
	var err error

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
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
	dbObject.ContentConnector.String = crypto.CreateRandomName()
	dbObject.EncryptIV = crypto.CreateIV()
	// Check if the user has permissions to update the ODObject
	var grant models.ODObjectPermission
	var ok bool
	if ok, grant = isUserAllowedToUpdateWithPermission(ctx, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		return ClassifyObjectACMError(err)
	}

	multipartReader, err := r.MultipartReader()
	if err != nil {
		return NewAppError(400, err, "unable to open multipart reader")
	}
	drainFunc, herr := h.acceptObjectUpload(ctx, multipartReader, &dbObject, &grant, false, nil)
	dp := ciphertext.FindCiphertextCacheByObject(&dbObject)
	if herr != nil {
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	masterKey := dp.GetMasterKey()

	modifiedACM := dbObject.RawAcm.String
	modifiedACM, err = aacAuth.GetFlattenedACM(modifiedACM)
	if err != nil {
		herr = ClassifyFlattenError(err)
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	dbObject.RawAcm = models.ToNullString(modifiedACM)
	modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(dbObject.OwnedBy.String, dbObject.Permissions, dbObject.RawAcm.String, dbObject.IsCreating())
	if err != nil {
		herr = NewAppError(500, err, err.Error())
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}
	dbObject.RawAcm = models.ToNullString(modifiedACM)
	dbObject.Permissions = modifiedPermissions
	// Final access check against altered ACM
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		herr = ClassifyObjectACMError(err)
		return abortUploadObject(logger, dp, &dbObject, true, herr)
	}

	// Verify user has access to change the share if the ACMs are different
	unchangedDbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr = NewAppError(500, err, err.Error())
	}
	// If the "share" or "f_share" parts have changed, then check that the
	// caller also has permission to share.
	if diff, herr := isAcmShareDifferent(dbObject.RawAcm.String, unchangedDbObject.RawAcm.String); herr != nil || diff {
		if herr != nil {
			return abortUploadObject(logger, dp, &dbObject, true, herr)
		}
		if !isUserAllowedToShare(ctx, &unchangedDbObject) {
			herr = NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to change the share for this object")
			return abortUploadObject(logger, dp, &dbObject, true, herr)
		}
	}

	consolidateChangingPermissions(&dbObject)
	// copy grant.EncryptKey to all existing permissions:
	for idx, permission := range dbObject.Permissions {
		models.CopyEncryptKey(masterKey, &grant, &permission)
		models.CopyEncryptKey(masterKey, &grant, &dbObject.Permissions[idx])
	}

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

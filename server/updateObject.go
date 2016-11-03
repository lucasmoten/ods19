package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/utils"
)

func (h AppServer) updateObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, _ := CallerFromContext(ctx)
	session := SessionIDFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)

	if r.Header.Get("Content-Type") != "application/json" {
		return NewAppError(400, nil, "expected application/json Content-Type")
	}

	requestObject, err = parseUpdateObjectRequestAsJSON(r, ctx)
	if err != nil {
		return NewAppError(400, err, "Error parsing JSON")
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

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

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, nil, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return NewAppError(405, nil, "The object cannot be modified because an ancestor is deleted.")
		case dbObject.IsDeleted:
			return NewAppError(405, nil, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
		}
	}

	// Check that assignment as deleted isn't occuring here. Should use deleteObject operations
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		return NewAppError(428, errors.New("Precondition required: Updating object as deleted not allowed. Send to trash or DELETE instead."), "Assigning object as deleted through update operation not allowed. Use deleteObject operation")
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		return NewAppError(428, errors.New("Precondition required: ChangeToken does not match expected value"), "ChangeToken does not match expected value. Object may have been changed by another request.")
	}

	// Retain existing value for parent.
	requestObject.ParentID = dbObject.ParentID

	// Retain existing values for content stream info
	requestObject.ContentConnector = dbObject.ContentConnector
	requestObject.ContentType = dbObject.ContentType
	requestObject.ContentSize = dbObject.ContentSize
	requestObject.ContentHash = dbObject.ContentHash
	requestObject.EncryptIV = dbObject.EncryptIV

	// Retain existing ownership
	requestObject.OwnedBy = models.ToNullString(dbObject.OwnedBy.String)

	// If there was no ACM provided...
	if len(requestObject.RawAcm.String) == 0 {
		// There was no change, retain existing from dbObject
		requestObject.RawAcm = models.ToNullString(dbObject.RawAcm.String)
	}

	// Assign existing permissions from the database object to the request object
	if len(requestObject.Permissions) == 0 {
		requestObject.Permissions = dbObject.Permissions
	} else {
		combinedPermissions := make([]models.ODObjectPermission, len(requestObject.Permissions)+len(dbObject.Permissions))
		// Any existing permissions will be marked as deleted, since past in overrides.
		idx := 0
		for _, d := range dbObject.Permissions {
			d.IsDeleted = true
			combinedPermissions[idx] = d
			idx = idx + 1
		}
		for _, r := range requestObject.Permissions {
			combinedPermissions[idx] = r
			idx = idx + 1
		}
		requestObject.Permissions = combinedPermissions
	}
	// Flatten ACM, then Normalize Read Permissions against ACM f_share
	if err = h.flattenACM(ctx, &requestObject); err != nil {
		return ClassifyFlattenError(err)
	}
	if herr := normalizeObjectReadPermissions(ctx, &requestObject); herr != nil {
		return herr
	}
	// Access check against altered ACM as a whole
	if err = h.isUserAllowedForObjectACM(ctx, &requestObject); err != nil {
		return ClassifyObjectACMError(err)
	}
	consolidateChangingPermissions(&requestObject)
	// copy grant.EncryptKey to all existing permissions:
	for idx, permission := range requestObject.Permissions {
		models.CopyEncryptKey(h.MasterKey, &grant, &permission)
		models.CopyEncryptKey(h.MasterKey, &grant, &requestObject.Permissions[idx])
	}

	// If ACM provided differs from what is currently set, then need to
	// Check AAC to compare user clearance to NEW metadata Classifications
	// to see if allowed for this user
	if strings.Compare(dbObject.RawAcm.String, requestObject.RawAcm.String) != 0 {
		// Ensure user is allowed this acm
		if err = h.isUserAllowedForObjectACM(ctx, &requestObject); err != nil {
			return ClassifyObjectACMError(err)
		}
		// If the "share" or "f_share" parts have changed, then check that the
		// caller also has permission to share.
		if diff, herr := isAcmShareDifferent(dbObject.RawAcm.String, requestObject.RawAcm.String); herr != nil || diff {
			if herr != nil {
				return herr
			}
			// Need to refetch dbObject as apparently the assignment of its permissions into request object is a reference instead of copy
			dbPermissions, _ := dao.GetPermissionsForObject(dbObject)
			dbObject.Permissions = dbPermissions
			if !isUserAllowedToShare(ctx, h.MasterKey, &dbObject) {
				return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to change the share for this object")
			}
		}

	}

	// Retain existing values from dbObject where no value was provided for key fields
	if len(requestObject.Name) == 0 {
		requestObject.Name = dbObject.Name
	}
	if len(requestObject.Description.String) == 0 {
		requestObject.Description.String = dbObject.Description.String
	}
	if len(requestObject.TypeName.String) == 0 {
		requestObject.TypeName.String = dbObject.TypeName.String
	}
	if len(requestObject.ContainsUSPersonsData) == 0 {
		requestObject.ContainsUSPersonsData = dbObject.ContainsUSPersonsData
	}
	if len(requestObject.ExemptFromFOIA) == 0 {
		requestObject.ExemptFromFOIA = dbObject.ExemptFromFOIA
	}

	// Call metadata connector to update the object in the data store
	// Force the modified by to be that of the caller
	requestObject.ModifiedBy = caller.DistinguishedName
	err = dao.UpdateObject(&requestObject)
	if err != nil {
		return NewAppError(500, err, "DAO Error updating object")
	}

	// After the update, check that key values have changed...
	if requestObject.ChangeCount <= dbObject.ChangeCount {
		if requestObject.ID == nil {
			log.Println("requestObject.ID = nil")
		}
		if dbObject.ID == nil {
			log.Println("dbObject.ID = nil")
		}
		log.Println(hex.EncodeToString(requestObject.ID))
		log.Println(hex.EncodeToString(dbObject.ID))

		msg := fmt.Sprintf("old changeCount: %d, new changeCount: %d, req id: %s, db id: %s", requestObject.ChangeCount, dbObject.ChangeCount, hex.EncodeToString(requestObject.ID), hex.EncodeToString(dbObject.ID))
		log.Println(msg)
		return NewAppError(500, nil, "ChangeCount didn't update when processing request "+msg)
	}
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
		msg := fmt.Sprintf("old token: %s, new token: %s", requestObject.ChangeToken, dbObject.ChangeToken)
		return NewAppError(500, nil, "ChangeToken didn't update when processing request "+msg)
	}

	dbObject, err = dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller))

	gem.Action = "update"
	gem.Payload = events.ObjectDriveEvent{
		ObjectID:     apiResponse.ID,
		ChangeToken:  apiResponse.ChangeToken,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: false,
		SessionID:    session,
	}
	h.EventQueue.Publish(gem)

	jsonResponse(w, apiResponse)
	return nil
}

func parseUpdateObjectRequestAsJSON(r *http.Request, ctx context.Context) (models.ODObject, error) {
	var jsonObject protocol.UpdateObjectRequest
	requestObject := models.ODObject{}
	var err error

	// Get ID from URI
	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		return requestObject, errors.New("Object Identifier in Request URI is not a hex string")
	}

	// Get portion from body
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return requestObject, err
	}

	if strings.Compare(hex.EncodeToString(requestObject.ID), jsonObject.ID) != 0 {
		return requestObject, errors.New("bad request: ID mismatch")
	}

	// Map changes over the requestObject
	if len(jsonObject.Name) > 0 {
		requestObject.Name = jsonObject.Name
	}
	requestObject.ChangeToken = jsonObject.ChangeToken
	if len(jsonObject.TypeName) > 0 {
		requestObject.TypeName = models.ToNullString(jsonObject.TypeName)
	}
	if len(jsonObject.Description) > 0 {
		requestObject.Description = models.ToNullString(jsonObject.Description)
	}
	convertedAcm, err := utils.MarshalInterfaceToString(jsonObject.RawAcm)
	if err != nil {
		return requestObject, err
	}
	if len(convertedAcm) > 0 {
		requestObject.RawAcm = models.ToNullString(convertedAcm)
	}
	requestObject.Permissions = mapping.MapPermissionToODPermissions(&jsonObject.Permission)
	if len(jsonObject.ContainsUSPersonsData) > 0 {
		requestObject.ContainsUSPersonsData = jsonObject.ContainsUSPersonsData
	}
	if len(jsonObject.ExemptFromFOIA) > 0 {
		requestObject.ExemptFromFOIA = jsonObject.ExemptFromFOIA
	}
	if len(jsonObject.Properties) > 0 {
		requestObject.Properties, err = mapping.MapPropertiesToODProperties(&jsonObject.Properties)
	}

	// Return it
	return requestObject, err
}

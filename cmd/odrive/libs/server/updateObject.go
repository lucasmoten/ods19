package server

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) updateObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	logger := LoggerFromContext(ctx)

	var requestObject models.ODObject
	var err error

	caller, _ := CallerFromContext(ctx)
	session := SessionIDFromContext(ctx)
	dao := DAOFromContext(ctx)

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
	hasAACAccessToOLDACM, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
	if err != nil {
		return NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccessToOLDACM {
		return NewAppError(403, nil, "Forbidden - User does not pass authorization checks for existing object ACM")
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return NewAppError(405, err, "The object cannot be modified because an ancestor is deleted.")
		case dbObject.IsDeleted:
			return NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
		}
	}

	// Check that assignment as deleted isn't occuring here. Should use deleteObject operations
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		return NewAppError(428, nil, "Assigning object as deleted through update operation not allowed. Use deleteObject operation")
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		return NewAppError(428, nil, "ChangeToken does not match expected value. Object may have been changed by another request.")
	}

	// Check that the parent of the object passed in matches the current state
	// of the object in the data store.
	if bytes.Compare(requestObject.ParentID, dbObject.ParentID) != 0 {
		return NewAppError(428, nil, "ParentID does not match expected value. Use moveObject to change this objects location.")
	}

	// Check that the owner of the object passed in matches the current state
	// of the object in the data store.
	if len(requestObject.OwnedBy.String) == 0 {
		requestObject.OwnedBy = models.ToNullString(dbObject.OwnedBy.String)
	}

	if strings.Compare(requestObject.OwnedBy.String, dbObject.OwnedBy.String) != 0 {
		return NewAppError(428, nil, "OwnedBy does not match expected value.  Use changeOwner to transfer ownership.")
	}

	// If there was no ACM provided...
	if len(requestObject.RawAcm.String) == 0 {
		// There was no change, retain existing from dbObject
		requestObject.RawAcm = models.ToNullString(dbObject.RawAcm.String)
	}

	// Assign existing permissions from the database object to the request object
	requestObject.Permissions = dbObject.Permissions
	// Flatten ACM, then Normalize Read Permissions against ACM f_share
	hasAACAccess := false
	err = h.flattenACM(logger, &requestObject)
	if err != nil {
		return NewAppError(400, err, "ACM provided could not be flattened")
	}
	if herr := normalizeObjectReadPermissions(ctx, &requestObject); herr != nil {
		return herr
	}
	// Access check against altered ACM as a whole
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

	// If ACM provided differs from what is currently set, then need to
	// Check AAC to compare user clearance to NEW metadata Classifications
	// to see if allowed for this user
	if strings.Compare(dbObject.RawAcm.String, requestObject.RawAcm.String) != 0 {
		// Ensure user is allowed this acm
		hasAACAccessToNewACM, err := h.isUserAllowedForObjectACM(ctx, &requestObject)
		if err != nil {
			return NewAppError(502, err, "Error communicating with authorization service")
		}
		if !hasAACAccessToNewACM {
			return NewAppError(403, err, "Forbidden - User does not pass authorization checks for new object ACM")
			//return NewAppError(403, err, "Unauthorized", zap.String("origination", "No access to new ACM on Update"), zap.String("acm", requestObject.RawAcm.String))
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

	apiResponse := mapping.MapODObjectToObject(&requestObject).WithCallerPermission(protocolCaller(caller))

	h.EventQueue.Publish(events.Index{
		ObjectID:     apiResponse.ID,
		Action:       "update",
		Timestamp:    time.Now().Format(time.RFC3339),
		ChangeToken:  apiResponse.ChangeToken,
		UserDN:       caller.DistinguishedName,
		StreamUpdate: false,
		SessionID:    session,
	})
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
		requestObject.TypeName.String = jsonObject.TypeName
		requestObject.TypeName.Valid = true
	}
	if len(jsonObject.Description) > 0 {
		requestObject.Description.String = jsonObject.Description
		requestObject.Description.Valid = true
	}
	convertedAcm, err := utils.MarshalInterfaceToString(jsonObject.RawAcm)
	if err != nil {
		return requestObject, err
	} else {
		if len(convertedAcm) > 0 {
			requestObject.RawAcm.String = convertedAcm
			requestObject.RawAcm.Valid = true
		}
	}
	if len(jsonObject.Properties) > 0 {
		requestObject.Properties, err = mapping.MapPropertiesToODProperties(&jsonObject.Properties)
	}

	// Return it
	return requestObject, err
}

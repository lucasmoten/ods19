package server

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func moveObjectRaw(dao dao.DAO, ctx context.Context, caller Caller, resourceStrings []string, aacAuth *auth.AACAuth, requestObject models.ODObject, dbObject *models.ODObject) (int, error, string) {

	// Capture and overwrite here for comparison later after the update
	requestObject.ChangeCount = dbObject.ChangeCount

	// Auth check
	if ok := isUserAllowedToUpdate(ctx, dbObject); !ok {
		return http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object"
	}
	if !aacAuth.IsUserOwner(caller.DistinguishedName, resourceStrings, dbObject.OwnedBy.String) {
		return http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User must be an object owner to move the object"
	}

	// Object state check
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return http.StatusGone, errors.New("Forbidden"), "The object no longer exists."
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return http.StatusMethodNotAllowed, errors.New("Forbidden"), "The object cannot be modified because an ancestor is deleted."
		case dbObject.IsDeleted:
			return http.StatusMethodNotAllowed, errors.New("Forbidden"), "The object is currently in the trash. Use removeObjectFromTrash to restore it"
		}
	}
	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		return http.StatusPreconditionRequired, errors.New("Precondition required: ChangeToken does not match expected value"), "ChangeToken does not match expected value. Object may have been changed by another request."
	}

	// Check that the parent of the object passed in is different then the current
	// state of the object in the data store
	if bytes.Compare(requestObject.ParentID, dbObject.ParentID) == 0 {
		// NOOP, will return current state
		requestObject = *dbObject
	} else {
		// Changing parent...
		// If making the parent something other then the root...
		if len(requestObject.ParentID) > 0 {

			targetParent := models.ODObject{}
			targetParent.ID = requestObject.ParentID
			// Look up the parent
			dbParent, err := dao.GetObject(targetParent, false)
			if err != nil {
				return http.StatusBadRequest, err, "Error retrieving parent to move object into"
			}
			// Check if the user has permission to create children under the target
			// object for which they are moving this one to (the parentID)
			if ok := isUserAllowedToCreate(ctx, &dbParent); !ok {
				return http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to move this object to target"
			}

			// Parent must not be deleted
			if targetParent.IsDeleted {
				if targetParent.IsExpunged {
					return http.StatusGone, errors.New("Forbidden"), "Unable to move object into an object that does not exist"
				}
				return http.StatusMethodNotAllowed, errors.New("Forbidden"), "Unable to move object into an object that is deleted"
			}

			// #60 Check that the parent being assigned for the object passed in does not
			// result in a circular reference
			if bytes.Compare(requestObject.ParentID, requestObject.ID) == 0 {
				return http.StatusBadRequest, err, "ParentID cannot be set to the ID of the object. Circular references are not allowed."
			}
			circular, err := dao.IsParentIDADescendent(requestObject.ID, requestObject.ParentID)
			if err != nil {
				return http.StatusInternalServerError, err, "Error retrieving ancestor to check for circular references"
			}
			if circular {
				return http.StatusBadRequest, errors.New("Forbidden"), "ParentID cannot be set to the value specified as would result in a circular reference"
			}
		}
	}

	// Call metadata connector to update the object in the data store
	// We reference the dbObject here instead of request to isolate what is
	// allowed to be changed in this operation
	// Force the modified by to be that of the caller
	dbObject.ModifiedBy = caller.DistinguishedName
	dbObject.ParentID = requestObject.ParentID
	err := dao.UpdateObject(dbObject)
	if err != nil {
		log.Printf("Error updating object: %v", err)
		return http.StatusInternalServerError, nil, "Error saving object in new location"
	}

	// After the update, check that key values have changed...
	if dbObject.ChangeCount <= requestObject.ChangeCount {
		return http.StatusInternalServerError, nil, "ChangeCount didn't update when processing move request"
	}
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
		return http.StatusInternalServerError, nil, "ChangeToken didn't update when processing move request"
	}
	return 0, nil, ""
}

func (h AppServer) moveObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	session := SessionIDFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	// Get object
	if r.Header.Get("Content-Type") != "application/json" {
		return NewAppError(http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
	}
	requestObject, err = parseMoveObjectRequestAsJSON(r, ctx)
	if err != nil {
		return NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
	}
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
	}

	code, errCause, msg := moveObjectRaw(
		dao,
		ctx,
		caller,
		getKnownResourceStringsFromUserGroups(ctx),
		aacAuth,
		requestObject,
		&dbObject,
	)

	if errCause != nil || msg != "" {
		return NewAppError(code, errCause, msg)
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

func parseMoveObjectRequestAsJSON(r *http.Request, ctx context.Context) (models.ODObject, error) {
	var jsonObject protocol.MoveObjectRequest
	var requestObject models.ODObject
	var err error

	// Depends on this for the changeToken
	err = util.FullDecode(r.Body, &jsonObject)
	if err != nil {
		return requestObject, err
	}

	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return requestObject, errors.New("Could not get capture groups")
	}

	// Initialize requestobject with the objectId being requested
	if captured["objectId"] == "" {
		return requestObject, errors.New("Could not extract ObjectID from URI")
	}
	_, err = hex.DecodeString(captured["objectId"])
	if err != nil {
		return requestObject, errors.New("Invalid ObjectID in URI")
	}
	jsonObject.ID = captured["objectId"]
	// And the new folderId
	if len(captured["folderId"]) > 0 {
		_, err = hex.DecodeString(captured["folderId"])
		if err != nil {
			return requestObject, errors.New("Invalid folderId in URI")
		}
		jsonObject.ParentID = captured["folderId"]
	}

	// Map to internal object type
	requestObject, err = mapping.MapMoveObjectRequestToODObject(&jsonObject)
	return requestObject, err
}

func getKnownResourceStringsFromUserGroups(ctx context.Context) (resourceStrings []string) {
	groups, ok := GroupsFromContext(ctx)
	if !ok {
		return resourceStrings
	}
	dao := DAOFromContext(ctx)
	acmGrantees, err := dao.GetAcmGrantees(groups)
	if err != nil {
		return resourceStrings
	}
	for _, acmGrantee := range acmGrantees {
		resourceStrings = append(resourceStrings, acmGrantee.ResourceName())
	}

	return resourceStrings
}

package server

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) moveObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	// Parse Request in sent format
	if r.Header.Get("Content-Type") != "application/json" {
		return NewAppError(http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
	}
	requestObject, err = parseMoveObjectRequestAsJSON(r, ctx)
	if err != nil {
		return NewAppError(400, err, "Error parsing JSON")
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Capture and overwrite here for comparison later after the update
	requestObject.ChangeCount = dbObject.ChangeCount

	// Check if the user has permissions to update the ODObject
	if ok := isUserAllowedToUpdate(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to update this object")
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

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		return NewAppError(428, nil, "ChangeToken does not match expected value. Object may have been changed by another request.")
	}

	// Check that the parent of the object passed in is different then the current
	// state of the object in the data store
	if bytes.Compare(requestObject.ParentID, dbObject.ParentID) == 0 {
		// NOOP, will return current state
		requestObject = dbObject
	} else {
		// Changing parent...
		// If making the parent something other then the root...
		if len(requestObject.ParentID) > 0 {
			targetParent := models.ODObject{}
			targetParent.ID = requestObject.ParentID
			// Look up the parent
			dbParent, err := dao.GetObject(targetParent, false)
			if err != nil {
				return NewAppError(400, err, "Error retrieving parent to move object into")
			}

			// Check if the user has permission to create children under the target
			// object for which they are moving this one to (the parentID)
			if ok := isUserAllowedToCreate(ctx, h.MasterKey, &dbParent); !ok {
				return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to move this object to target")
			}

			// Parent must not be deleted
			if targetParent.IsDeleted {
				if targetParent.IsExpunged {
					return NewAppError(410, err, "Unable to move object into an object that does not exist")
				}
				return NewAppError(405, err, "Unable to move object into an object that is deleted")
			}

			// #60 Check that the parent being assigned for the object passed in does not
			// result in a circular reference
			if bytes.Compare(requestObject.ParentID, requestObject.ID) == 0 {
				return NewAppError(400, err, "ParentID cannot be set to the ID of the object. Circular references are not allowed.")
			}
			circular, err := dao.IsParentIDADescendent(requestObject.ID, requestObject.ParentID)
			if err != nil {
				return NewAppError(500, err, "Error retrieving ancestor to check for circular references")
			}
			if circular {
				return NewAppError(400, err, "ParentID cannot be set to the value specified as would result in a circular reference")
			}

		}

		// Call metadata connector to update the object in the data store
		// We reference the dbObject here instead of request to isolate what is
		// allowed to be changed in this operation
		// Force the modified by to be that of the caller
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ParentID = requestObject.ParentID
		err := dao.UpdateObject(&dbObject)
		if err != nil {
			log.Printf("Error updating object: %v", err)
			return NewAppError(500, nil, "Error saving object in new location")
		}

		// After the update, check that key values have changed...
		if dbObject.ChangeCount <= requestObject.ChangeCount {
			return NewAppError(500, nil, "ChangeCount didn't update when processing move request")
		}
		if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
			return NewAppError(500, nil, "ChangeToken didn't update when processing move request")
		}
	}

	apiResponse := mapping.MapODObjectToObject(&dbObject)
	jsonResponse(w, apiResponse)

	return nil
}

func parseMoveObjectRequestAsJSON(r *http.Request, ctx context.Context) (models.ODObject, error) {
	var jsonObject protocol.Object
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
			return requestObject, errors.New("Invalid flderId in URI")
		}
		jsonObject.ParentID = captured["folderId"]
	}

	// Map to internal object type
	requestObject, err = mapping.MapObjectToODObject(&jsonObject)
	return requestObject, err
}

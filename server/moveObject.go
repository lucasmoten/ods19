package server

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/events"
	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

func moveObjectRaw(ctx context.Context, dao dao.DAO, caller Caller, resourceStrings []string, aacAuth *auth.AACAuth, requestObject models.ODObject, dbObject *models.ODObject) (int, string, error) {

	// Capture and overwrite here for comparison later after the update
	requestObject.ChangeCount = dbObject.ChangeCount

	// Auth check
	if ok := isUserAllowedToUpdate(ctx, dbObject); !ok {
		return http.StatusForbidden, "Forbidden - User does not have permission to update this object", errors.New("Forbidden")
	}
	if !aacAuth.IsUserOwner(caller.DistinguishedName, resourceStrings, dbObject.OwnedBy.String) {
		return http.StatusForbidden, "Forbidden - User must be an object owner to move the object", errors.New("Forbidden")
	}

	// Object state check
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return http.StatusGone, "The object no longer exists.", errors.New("Forbidden")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return http.StatusBadRequest, "The object cannot be modified because an ancestor is deleted.", errors.New("Forbidden")
		case dbObject.IsDeleted:
			return http.StatusBadRequest, "The object is currently in the trash. Use removeObjectFromTrash to restore it", errors.New("Forbidden")
		}
	}
	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		return http.StatusBadRequest, "ChangeToken does not match expected value. Object may have been changed by another request.", errors.New("Precondition required: ChangeToken does not match expected value")
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
				return http.StatusBadRequest, "Error retrieving parent to move object into", err
			}
			// Check if the user has permission to create children under the target
			// object for which they are moving this one to (the parentID)
			if ok := isUserAllowedToCreate(ctx, &dbParent); !ok {
				return http.StatusForbidden, "Forbidden - User does not have permission to move this object to target", errors.New("Forbidden")
			}

			// Parent must not be deleted
			if targetParent.IsDeleted {
				if targetParent.IsExpunged {
					return http.StatusGone, "Unable to move object into an object that does not exist", errors.New("Forbidden")
				}
				return http.StatusBadRequest, "Unable to move object into an object that is deleted", errors.New("Forbidden")
			}

			// #60 Check that the parent being assigned for the object passed in does not
			// result in a circular reference
			if bytes.Compare(requestObject.ParentID, requestObject.ID) == 0 {
				return http.StatusBadRequest, "ParentID cannot be set to the ID of the object. Circular references are not allowed.", err
			}
			circular, err := dao.IsParentIDADescendent(requestObject.ID, requestObject.ParentID)
			if err != nil {
				return http.StatusInternalServerError, "Error retrieving ancestor to check for circular references", err
			}
			if circular {
				return http.StatusBadRequest, "ParentID cannot be set to the value specified as would result in a circular reference", errors.New("Forbidden")
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
		return http.StatusInternalServerError, "Error saving object in new location", nil
	}

	// After the update, check that key values have changed...
	if dbObject.ChangeCount <= requestObject.ChangeCount {
		return http.StatusInternalServerError, "ChangeCount didn't update when processing move request", nil
	}
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
		return http.StatusInternalServerError, "ChangeToken didn't update when processing move request", nil
	}
	return 0, "", nil
}

func (h AppServer) moveObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MOVE")

	// Get object
	if !util.IsApplicationJSON(r.Header.Get("Content-Type")) {
		herr := NewAppError(http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		h.publishError(gem, herr)
		return herr
	}
	requestObject, err = parseMoveObjectRequestAsJSON(ctx, r)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing JSON")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)

	code, msg, errCause := moveObjectRaw(
		ctx,
		dao,
		caller,
		getKnownResourceStringsFromUserGroups(ctx),
		aacAuth,
		requestObject,
		&dbObject,
	)

	if errCause != nil || msg != "" {
		herr := NewAppError(code, errCause, msg)
		h.publishError(gem, herr)
		return herr
	}

	parents, err := dao.GetParents(dbObject)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "error retrieving object parents")
		h.publishError(gem, herr)
		return herr
	}

	filtered := redactParents(ctx, aacAuth, parents)
	if appError := errOnDeletedParents(parents); appError != nil {
		h.publishError(gem, appError)
		return appError
	}
	crumbs := breadcrumbsFromParents(filtered)

	auditModified := NewResourceFromObject(dbObject)

	apiResponse := mapping.MapODObjectToObject(&dbObject).WithCallerPermission(protocolCaller(caller)).WithBreadcrumbs(crumbs)

	gem.Payload.ChangeToken = apiResponse.ChangeToken
	gem.Payload.Audit = audit.WithModifiedPairList(gem.Payload.Audit, audit.NewModifiedResourcePair(auditOriginal, auditModified))
	gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

func parseMoveObjectRequestAsJSON(ctx context.Context, r *http.Request) (models.ODObject, error) {
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
		log.Printf("Error calling GetAcmGrantees %s", err.Error())
		return resourceStrings
	}
	for _, acmGrantee := range acmGrantees {
		resourceStrings = append(resourceStrings, acmGrantee.ResourceName())
	}

	return resourceStrings
}

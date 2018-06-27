package server

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/events"
	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/services/audit"
)

func (h AppServer) restoreVersion(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)

	gem.Action = "update"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MODIFY")

	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusInternalServerError, errors.New("Could not get capture groups"), "No capture groups.")
		h.publishError(gem, herr)
		return herr
	}

	if captured["objectId"] == "" {
		herr := NewAppError(http.StatusBadRequest, errors.New("Could not extract objectID from URI"), "URI: "+r.URL.Path)
		h.publishError(gem, herr)
		return herr
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Invalid objectID in URI.")
		h.publishError(gem, herr)
		return herr
	}
	requestObject.ID = bytesObjectID
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))
	if captured["revisionId"] == "" {
		herr := NewAppError(http.StatusBadRequest, errors.New("Could not extract revisionId from URI"), "URI: "+r.URL.Path)
		h.publishError(gem, herr)
		return herr
	}
	requestObject.ChangeCount, err = strconv.Atoi(captured["revisionId"])
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Invalid revisionId in URI.")
		h.publishError(gem, herr)
		return herr
	}
	// Get the change token off the request.
	changeToken, err := protocol.NewChangeTokenStructFromJSONBody(r.Body)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Unexpected change token")
		h.publishError(gem, herr)
		return herr
	}
	requestObject.ChangeToken = changeToken.ChangeToken
	var herr *AppError
	// Current version
	// - retrieve
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	auditOriginal := NewResourceFromObject(dbObject)
	// - auth check to read, and not deleted
	if herr, _ = getFileKeyAndCheckAuthAndObjectState(ctx, h, &dbObject); herr != nil {
		h.publishError(gem, herr)
		return herr
	}
	// - auth check to acm
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		herr := NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return herr
	}
	// - auth check to update
	var grant models.ODObjectPermission
	if ok, grant = isUserAllowedToUpdateWithPermission(ctx, &dbObject); !ok {
		herr := NewAppError(http.StatusForbidden, errors.New("forbidden"), "user does not have permission to update this object")
		h.publishError(gem, herr)
		return herr
	}
	// - changeToken check
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		herr := NewAppError(http.StatusBadRequest, errors.New("Bad request: ChangeToken does not match expected value"), "ChangeToken does not match expected value. Object may have been changed by another request.")
		h.publishError(gem, herr)
		return herr
	}
	// Requested revision
	// -retrieve
	dbObjectRevision, err := dao.GetObjectRevision(requestObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	// - auth check to read, and not deleted
	if herr, _ = getFileKeyAndCheckAuthAndObjectState(ctx, h, &dbObjectRevision); herr != nil {
		h.publishError(gem, herr)
		return herr
	}
	// - auth check to acm
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObjectRevision.RawAcm.String); err != nil {
		herr := NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return herr
	}

	// Parents must match
	if bytes.Compare(dbObject.ParentID, dbObjectRevision.ParentID) != 0 {
		herr := NewAppError(http.StatusBadRequest, errors.New("Bad request"), "Bad request - Cannot restore a revision of an object if it is located in a different folder then the current revision.  Move the object as owner, then restore.")
		h.publishError(gem, herr)
		return herr
	}
	// Owner must match
	if dbObject.OwnedBy != dbObjectRevision.OwnedBy {
		herr := NewAppError(http.StatusBadRequest, errors.New("Bad request"), "Bad request - Cannot restore a revision of an object if the owner of the revision is different then the current revision.")
		h.publishError(gem, herr)
		return herr
	}

	// Check if version to assign is already current
	sameversion := (requestObject.ChangeCount == dbObject.ChangeCount)
	if !sameversion {
		// Modify for update, bring forward values
		dbObject.ModifiedBy = caller.DistinguishedName
		// - first class fields
		dbObject.ContainsUSPersonsData = dbObjectRevision.ContainsUSPersonsData
		dbObject.ContentType = dbObjectRevision.ContentType
		dbObject.Description = dbObjectRevision.Description
		dbObject.ExemptFromFOIA = dbObjectRevision.ExemptFromFOIA
		dbObject.Name = dbObjectRevision.Name
		dbObject.TypeID = dbObjectRevision.TypeID
		dbObject.TypeName = dbObjectRevision.TypeName
		// - file stream
		dbObject.ContentConnector = dbObjectRevision.ContentConnector
		dbObject.ContentHash = dbObjectRevision.ContentHash
		dbObject.ContentSize = dbObjectRevision.ContentSize
		dbObject.EncryptIV = dbObjectRevision.EncryptIV
		// - properties: delete current by clearing value before saving
		for cpi := range dbObject.Properties {
			dbObject.Properties[cpi].Value = models.ToNullString("")
		}
		// - properties: add from revision
		for _, rproperty := range dbObjectRevision.Properties {
			rproperty.IsDeleted = false
			rproperty.ID = nil
			dbObject.Properties = append(dbObject.Properties, rproperty)
		}
		// - acm & permissions remain the same
		// - copy grant.EncryptKey to all permissions:
		masterKey := ciphertext.FindCiphertextCacheByObject(nil).GetMasterKey()
		for idx, permission := range dbObject.Permissions {
			models.CopyEncryptKey(masterKey, &grant, &permission)
			models.CopyEncryptKey(masterKey, &grant, &dbObject.Permissions[idx])
		}

		// Apply changes to Data Access Layer
		if err := dao.UpdateObject(&dbObject); err != nil {
			herr := NewAppError(http.StatusInternalServerError, err, "DAO Error updating object")
			h.publishError(gem, herr)
			return herr
		}

		// Retrieve for response and event publishing
		dbObject, err = dao.GetObject(requestObject, true)
		if err != nil {
			herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
			h.publishError(gem, herr)
			return herr
		}
	} else {
		// For sameversion, no modification will occur
		// Change the GEM to reflect this is Access.
		gem.Action = "access"
		gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
		gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")
	} //sameversion

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
	gem.Payload.StreamUpdate = false
	gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)

	return nil
}

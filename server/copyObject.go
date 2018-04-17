package server

import (
	"encoding/hex"
	"errors"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/events"
	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/services/audit"
)

func (h AppServer) copyObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "create"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventCreate")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "CREATE")

	requestObject, err := parseGetObjectRequest(ctx)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing URI")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		code, msg, err := getObjectDAOError(err)
		herr := NewAppError(code, err, msg)
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(dbObject))
	gem.Payload.ChangeToken = dbObject.ChangeToken

	// Check if the user has permissions to read the ODObject
	//		Permission.grantee matches caller, and AllowRead is true
	ok, existingPerm := isUserAllowedToReadWithPermission(ctx, &dbObject)
	if !ok {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to read/view this object")
		h.publishError(gem, herr)
		return herr
	}
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		herr := NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return herr
	}

	if ok, code, err := isExpungedOrAnscestorDeletedErr(dbObject); !ok {
		herr := NewAppError(code, err, "expunged or ancesor deleted")
		h.publishError(gem, herr)
		return herr
	}

	if dbObject.IsDeleted {
		apiResponse := mapping.MapODObjectToDeletedObject(&dbObject).WithCallerPermission(protocolCaller(caller))
		jsonResponse(w, apiResponse)
		h.publishSuccess(gem, w)
		return nil
	}

	// Get revisions that will be copied

	// Snippets
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusBadGateway, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
		h.publishError(gem, herr)
		return herr
	}
	user.Snippets = snippetFields

	// -- initialize paging request as the object id
	pagingRequest := protocol.PagingRequest{ObjectID: hex.EncodeToString(requestObject.ID)}
	// change sort to be in order of revisions
	pagingRequest.SortSettings = []protocol.SortSetting{protocol.SortSetting{SortField: "changecount", SortAscending: true}}
	// get them
	response, err := dao.GetObjectRevisionsByUser(user, mapping.MapPagingRequestToDAOPagingRequest(&pagingRequest), dbObject, true)
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "General error")
		h.publishError(gem, herr)
		return herr
	}

	// Process revisions
	var apiResponse protocol.Object
	var copiedObject models.ODObject
	for _, o := range response.Objects {
		if isAllowed, _ := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, o.RawAcm.String); isAllowed {
			o.CreatedBy = caller.DistinguishedName
			o.ModifiedBy = caller.DistinguishedName
			o.OwnedBy = models.ToNullString("user/" + caller.DistinguishedName)
			o.ID = copiedObject.ID                   // will be empty until created
			o.ChangeToken = copiedObject.ChangeToken // will be empty until created
			if len(copiedObject.ID) > 0 {
				// update
				// - reset gem
				gem = ResetBulkItem(gem)
				gem.Action = "update"
				gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventModify")
				gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "MODIFY")
				// - permissions retained for every revision of the copy
				o.Permissions = copiedObject.Permissions
				// - save metadata
				err = dao.UpdateObject(&o)
				if err != nil {
					herr := NewAppError(http.StatusInternalServerError, err, "error storing object")
					h.publishError(gem, herr)
					return herr
				}
				// - copy result back into copiedObject
				copiedObject = o
				// - gem success
				apiResponse = mapping.MapODObjectToObject(&copiedObject)
				auditResource := NewResourceFromObject(copiedObject)
				gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(copiedObject.ID))
				gem.Payload.ObjectID = apiResponse.ID
				gem.Payload.ChangeToken = copiedObject.ChangeToken
				gem.Payload.StreamUpdate = false
				gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, auditResource)
				gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
				h.publishSuccess(gem, w)
			} else {
				// create
				// - permissions for all revisions of the copy are the same because getting revisions always
				//   returns the current ermissions
				// Owner gets full cruds
				perm, err := models.CreateODPermissionFromResource(o.OwnedBy.String)
				perm.AllowCreate, perm.AllowRead, perm.AllowUpdate, perm.AllowDelete, perm.AllowShare = true, true, true, true, true
				masterKey := ciphertext.FindCiphertextCacheByObject(&o).GetMasterKey()
				models.CopyEncryptKey(masterKey, &existingPerm, &perm)
				o.Permissions = append(o.Permissions, perm)
				modifiedACM, err := aacAuth.InjectPermissionsIntoACM(o.Permissions, o.RawAcm.String)
				if err != nil {
					logger.Error("cannot inject permissions into copied object", zap.Error(err))
					continue
				}
				modifiedPermissions, modifiedACM, err := aacAuth.NormalizePermissionsFromACM(o.OwnedBy.String, o.Permissions, modifiedACM, false)
				if err != nil {
					logger.Error("error calling NormalizePermissionsFromACM", zap.Error(err))
					continue
				}
				o.RawAcm = models.ToNullString(modifiedACM)
				o.Permissions = modifiedPermissions
				// - save metadata
				copiedObject, err = dao.CreateObject(&o)
				if err != nil {
					herr := NewAppError(http.StatusInternalServerError, err, "error storing object")
					h.publishError(gem, herr)
					return herr
				}
				// - gem success
				apiResponse = mapping.MapODObjectToObject(&copiedObject)
				auditResource := NewResourceFromObject(copiedObject)
				gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(copiedObject.ID))
				gem.Payload.ObjectID = apiResponse.ID
				gem.Payload.ChangeToken = copiedObject.ChangeToken
				gem.Payload.StreamUpdate = false
				gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, auditResource)
				gem.Payload = events.WithEnrichedPayload(gem.Payload, apiResponse)
				h.publishSuccess(gem, w)
			}

		}
	}

	parents, err := dao.GetParents(copiedObject)
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

	apiResponse = mapping.MapODObjectToObject(&copiedObject).
		WithCallerPermission(protocolCaller(caller)).
		WithBreadcrumbs(crumbs)
	jsonResponse(w, apiResponse)

	h.publishSuccess(gem, w)

	return nil
}

package server

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"bitbucket.di2e.net/dime/object-drive-server/mapping"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
)

// listObjects returns a paged object result set. If parentID is given in the request URI,
// then it is used to list the children within it, otherwise, the root for the given user
// is listed. For a user, the root is defined as those objects that they own
// which have no parent identifier.
func (h AppServer) listObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	gem, _ := GEMFromContext(ctx)
	gem.Action = "list"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	parentObject := models.ODObject{}
	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err = protocol.NewPagingRequest(r, captured, false)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing request")
		h.publishError(gem, herr)
		return herr
	}

	parentObject, err = assignObjectIDFromPagingRequest(pagingRequest, parentObject)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Object Identifier in Request URI is not a hex string")
		h.publishError(gem, herr)
		return herr
	}

	// TODO can we remove this? We should expect snippets to be set by now.
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusBadGateway, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
		h.publishError(gem, herr)
		return herr

	}
	user.Snippets = snippetFields

	// Fetch the matching objects
	var results models.ODObjectResultset
	if parentObject.ID == nil {
		// Requesting root
		results, err = dao.GetRootObjectsWithPropertiesByUser(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest))
	} else {
		// Requesting children of an object. Load parent first.
		dbObject, err := dao.GetObject(parentObject, false)
		if err != nil {
			log.Println(err)
			code, msg := listObjectsDAOErr(err)
			herr := NewAppError(code, err, msg)
			h.publishError(gem, herr)
			return herr

		}
		// Check for permission to read this object
		if ok := isUserAllowedToRead(ctx, &dbObject); !ok {
			herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to list contents of this object")
			h.publishError(gem, herr)
			return herr
		}

		// Check if deleted
		if ok, code, err := isDeletedErr(dbObject); !ok {
			herr := NewAppError(code, err, "deleted object")
			h.publishError(gem, herr)
			return herr
		}

		// Get the objects
		results, err = dao.GetChildObjectsWithPropertiesByUser(user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest), parentObject)

	}
	if err != nil {
		code, msg := listObjectsDAOErr(err)
		herr := NewAppError(code, err, msg)
		h.publishError(gem, herr)
		return herr
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)

	// Caller permissions
	for objectIndex, object := range apiResponse.Objects {
		apiResponse.Objects[objectIndex] = object.WithCallerPermission(protocolCaller(caller))
	}

	gem.Payload.Audit = WithResourcesFromResultset(gem.Payload.Audit, results)

	// Output as JSON
	jsonResponse(w, apiResponse)
	h.publishSuccess(gem, w)
	return nil
}

func assignObjectIDFromPagingRequest(pagingRequest *protocol.PagingRequest, parent models.ODObject) (models.ODObject, error) {
	var err error
	if len(pagingRequest.ObjectID) == 0 {
		parent.ID = nil
	} else {
		parent.ID, err = hex.DecodeString(pagingRequest.ObjectID)
	}
	return parent, err
}

func isDeletedErr(obj models.ODObject) (ok bool, code int, err error) {
	switch {
	case obj.IsExpunged:
		return false, http.StatusGone, errors.New("object is expunged")
	case obj.IsAncestorDeleted:
		return false, http.StatusConflict, errors.New("object ancestor is deleted")
	case obj.IsDeleted:
		return false, http.StatusConflict, errors.New("object is deleted")
	}
	return true, 0, nil
}

func listObjectsDAOErr(err error) (code int, message string) {
	switch err {
	case sql.ErrNoRows:
		return http.StatusBadRequest, "Object not found"
	default:
		return http.StatusInternalServerError, "Error retrieving object"
	}
}

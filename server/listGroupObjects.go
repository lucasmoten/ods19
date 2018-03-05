package server

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"strings"

	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/services/audit"
)

// listGroupObjects returns a paged object result set of those objects owned by a specified group.
// The calling user must be a member of teh group.
func (h AppServer) listGroupObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)

	gem, _ := GEMFromContext(ctx)
	gem.Action = "list"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventSearchQry")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "PARAMETER_SEARCH")
	gem.Payload.Audit = audit.WithQueryString(gem.Payload.Audit, r.URL.String())

	snippetFields, _ := SnippetsFromContext(ctx)
	user.Snippets = snippetFields

	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err = protocol.NewPagingRequest(r, captured, false)
	if err != nil {
		herr := NewAppError(400, err, "Error parsing request")
		h.publishError(gem, herr)
		return herr
	}

	// Group name validation
	groupName := captured["groupName"]
	if groupName == "" {
		msg := "Group name required when listing objects for group"
		herr := NewAppError(400, fmt.Errorf(msg), msg)
		h.publishError(gem, herr)
		return herr
	}
	groupName = strings.ToLower(groupName)

	// Fetch the matching objects
	var results models.ODObjectResultset
	results, err = dao.GetRootObjectsWithPropertiesByGroup(groupName, user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest))
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

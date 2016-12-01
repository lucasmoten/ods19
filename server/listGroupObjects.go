package server

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"strings"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// listGroupObjects returns a paged object result set of those objects owned by a specified group.
// The calling user must be a member of teh group.
func (h AppServer) listGroupObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	caller, _ := CallerFromContext(ctx)
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)
	snippetFields, _ := SnippetsFromContext(ctx)
	user.Snippets = snippetFields

	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err = protocol.NewPagingRequest(r, captured, false)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Group name validation
	groupName := captured["groupName"]
	if groupName == "" {
		msg := "Group name required when listing objects for group"
		return NewAppError(400, fmt.Errorf(msg), msg)
	}
	groupName = strings.ToLower(groupName)
	userHasGroup := false
	for _, group := range caller.Groups {
		if strings.ToLower(group) == groupName {
			userHasGroup = true
			break
		}
	}
	if !userHasGroup {
		msg := "Forbidden. Not a member of requested group"
		return NewAppError(403, fmt.Errorf(msg), msg)
	}

	// Fetch the matching objects
	var results models.ODObjectResultset
	results, err = dao.GetRootObjectsWithPropertiesByGroup(groupName, user, mapping.MapPagingRequestToDAOPagingRequest(pagingRequest))
	if err != nil {
		code, msg := listObjectsDAOErr(err)
		return NewAppError(code, err, msg)
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&results)

	// Caller permissions
	for objectIndex, object := range apiResponse.Objects {
		apiResponse.Objects[objectIndex] = object.WithCallerPermission(protocolCaller(caller))
	}

	// Output as JSON
	jsonResponse(w, apiResponse)
	return nil
}

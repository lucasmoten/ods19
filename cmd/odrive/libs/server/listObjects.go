package server

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

// listObjects reuturns a paged object result set.  If parentID is given in the request URI,
// then it is used to list the children within it, otherwise, the root for the given user
// is listed. For a user, the root is defined as those objects that they own
// which have no parent identifier.
func (h AppServer) listObjects(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}
	dao := DAOFromContext(ctx)

	parentObject := models.ODObject{}
	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err = protocol.NewPagingRequest(r, captured, false)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	parentObject, err = assignObjectIDFromPagingRequest(pagingRequest, parentObject)
	if err != nil {
		return NewAppError(400, err, "Object Identifier in Request URI is not a hex string")
	}

	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		return NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
	}
	user.Snippets = snippetFields

	// Fetch the matching objects
	var response models.ODObjectResultset
	if parentObject.ID == nil {
		// Requesting root
		response, err = dao.GetRootObjectsWithPropertiesByUser(user, *pagingRequest)
	} else {
		// Requesting children of an object. Load parent first.
		dbObject, err := dao.GetObject(parentObject, false)
		if err != nil {
			log.Println(err)
			code, msg := listObjectsDAOErr(err)
			return NewAppError(code, err, msg)
		}
		// Check for permission to read this object
		if ok, _ := isUserAllowedToRead(ctx, &dbObject); !ok {
			return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to list contents of this object")
		}

		// Check if deleted
		if ok, code, err := isDeletedErr(dbObject); !ok {
			return NewAppError(code, err, "deleted object")
		}

		// Get the objects
		response, err = dao.GetChildObjectsWithPropertiesByUser(user, *pagingRequest, parentObject)

	}
	if err != nil {
		code, msg := listObjectsDAOErr(err)
		return NewAppError(code, err, msg)
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
	writeResultsetAsJSON(w, &apiResponse)
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

func writeResultsetAsJSON(w http.ResponseWriter, resp *protocol.ObjectResultset) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

func isDeletedErr(obj models.ODObject) (ok bool, code int, err error) {
	switch {
	case obj.IsExpunged:
		return false, 410, errors.New("object is expunged")
	case obj.IsAncestorDeleted:
		return false, 405, errors.New("object ancestor is deleted.")
	case obj.IsDeleted:
		return false, 405, errors.New("object is deleted")
	}
	return true, 0, nil
}

func listObjectsDAOErr(err error) (code int, message string) {
	switch err {
	case sql.ErrNoRows:
		return 404, "Object not found"
	default:
		return 500, "Error retrieving object"
	}
}

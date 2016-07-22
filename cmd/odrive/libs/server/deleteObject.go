package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) deleteObject(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}
	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		return NewAppError(502, errors.New("Error retrieving user permissions"), "Error communicating with upstream")
	}
	user.Snippets = snippetFields
	dao := DAOFromContext(ctx)

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		return NewAppError(400, err, "Error parsing JSON")
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Check if the user has permissions to delete the ODObject
	if ok := isUserAllowedToDelete(ctx, h.MasterKey, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to delete this object")
	}

	// If the object is already deleted,
	if dbObject.IsDeleted {
		// Check its state
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The referenced object no longer exists.")
		default:
			// NO change will be applied, but deletedDate will still be exposed in
			// the output
		}
	} else {
		// Call DAO to update the object to reflect that it is
		// deleted.  The DAO checks the changeToken and handles the child calls
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ChangeToken = requestObject.ChangeToken
		err = dao.DeleteObject(user, dbObject, true)
		if err != nil {
			return NewAppError(500, err, "DAO Error deleting object")
		}
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToDeletedObjectResponse(&dbObject)
	herr := deleteObjectResponse(w, r, &apiResponse)
	if herr != nil {
		return herr
	}
	return nil
}

// This same handler is used for both deleting an object (POST as new state), or deleting forever (DELETE)
func parseDeleteObjectRequest(r *http.Request, ctx context.Context) (models.ODObject, error) {
	// TODO: Create and Change to DeletedObjectRequest
	var jsonObject protocol.Object
	var requestObject models.ODObject
	var err error

	// Capture changeToken
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = util.FullDecode(r.Body, &jsonObject)
		if err != nil {
			return requestObject, err
		}
	}
	// Map to internal object type.
	requestObject, err = mapping.MapObjectToODObject(&jsonObject)
	if err != nil {
		return requestObject, err
	}

	// Extract object ID from the URI and map over the request object being sent back
	uriObject, err := parseGetObjectRequest(ctx)
	if err != nil {
		return requestObject, err
	}
	requestObject.ID = uriObject.ID

	// Ready
	return requestObject, err
}

func deleteObjectResponse(
	w http.ResponseWriter,
	r *http.Request,
	response *protocol.DeletedObjectResponse,
) *AppError {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		msg := "Error marshalling response as JSON"
		return NewAppError(500, err, msg)
	}
	w.Write(jsonData)
	return nil
}

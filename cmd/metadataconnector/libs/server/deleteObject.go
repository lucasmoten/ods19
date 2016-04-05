package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) deleteObject(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	var requestObject models.ODObject
	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing JSON")
		return
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObject(requestObject, false)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error retrieving object")
		return
	}

	// Check if the user has permissions to delete the ODObject
	//		Permission.grantee matches caller, and AllowDelete is true
	authorizedToDelete := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName &&
			permission.AllowDelete {
			authorizedToDelete = true
			break
		}
	}
	if !authorizedToDelete {
		sendErrorResponse(&w, 403, nil, "Unauthorized")
		return
	}

	// If the object is already deleted,
	if dbObject.IsDeleted {
		// Check its state
		switch {
		case dbObject.IsExpunged:
			sendErrorResponse(&w, 410, err, "The referenced object no longer exists.")
			return
		default:
			// NO change will be applied, but deletedDate will still be exposed in
			// the output
		}
	} else {
		// Call DAO to update the object to reflect that it is
		// deleted.  The DAO checks the changeToken and handles the child calls
		dbObject.ModifiedBy = caller.DistinguishedName
		dbObject.ChangeToken = requestObject.ChangeToken
		err = h.DAO.DeleteObject(dbObject, true)
		if err != nil {
			sendErrorResponse(&w, 500, err, "DAO Error deleting object")
			return
		}
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToDeletedObjectResponse(&dbObject)
	herr := deleteObjectResponse(w, r, &apiResponse)
	if herr != nil {
		sendAppErrorResponse(&w, herr)
		return
	}
	countOKResponse()
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
		err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
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
		log.Printf("Error marshalling response as json: %s", err.Error())
		msg := "Error marshalling response as JSON"
		return NewAppError(500, err, msg)
	}
	w.Write(jsonData)
	return nil
}

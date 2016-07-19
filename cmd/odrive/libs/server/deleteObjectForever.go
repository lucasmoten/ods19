package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) deleteObjectForever(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	var requestObject models.ODObject
	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectRequest(r, ctx)
	if err != nil {
		return NewAppError(400, err, "Error parsing JSON")
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Check if the user has permissions to delete the ODObject
	//		Permission.grantee matches caller, and AllowDelete is true
	if ok, _ := isUserAllowedToDelete(ctx, &dbObject); !ok {
		return NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to expunge this object")
	}

	// If the object is already expunged,
	if dbObject.IsExpunged {
		return NewAppError(410, err, "The referenced object no longer exists.")
	}

	// Call metadata connector to update the object to reflect that it is
	// expunged.  The DAO checks the changeToken and handles the child calls
	dbObject.ModifiedBy = caller.DistinguishedName
	dbObject.ChangeToken = requestObject.ChangeToken
	err = dao.ExpungeObject(dbObject, true)
	if err != nil {
		return NewAppError(500, err, "DAO Error expunging object")
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToExpungedObjectResponse(&dbObject)
	herr := deleteObjectForeverResponse(w, r, caller, &apiResponse)
	if herr != nil {
		return herr
	}
	return nil
}

func deleteObjectForeverResponse(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.ExpungedObjectResponse,
) *AppError {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return NewAppError(500, err, "cant marshal json")
	}
	w.Write(jsonData)
	return nil
}

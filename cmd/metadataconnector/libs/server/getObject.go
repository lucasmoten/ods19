package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/dao"
	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) getObject(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	var requestObject models.ODObject
	var err error

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error parsing URI")
		return
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		switch err {
		case dao.ErrMissingID:
			sendErrorResponse(&w, 400, err, "Must provide ID field")
		default:
			log.Println("Default error")
			sendErrorResponse(&w, 500, err, "Error retrieving object")
		}
		return
	}

	// Check if the user has permissions to read the ODObject
	//		Permission.grantee matches caller, and AllowRead is true
	authorizedToRead := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowRead {
			authorizedToRead = true
		}
	}
	if !authorizedToRead {
		sendErrorResponse(&w, 403, errors.New("Unauthorized"), "Unauthorized")
		return
	}

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error communicating with authorization service")
		return
	}
	if !hasAACAccess {
		sendErrorResponse(&w, 403, nil, "Unauthorized")
		return
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			sendErrorResponse(&w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted:
			sendErrorResponse(&w, 405, err, "The object cannot be retreived because an ancestor is deleted.")
			return
		}
	}

	// Response
	err = getObjectResponse(w, r, caller, &dbObject)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Unable to get object")
		return
	}
	countOKResponse()
}

func parseGetObjectRequest(ctx context.Context) (models.ODObject, error) {
	var requestObject models.ODObject

	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return requestObject, errors.New("Could not get capture groups")
	}

	// Initialize requestobject with the objectId being requested
	if captured["objectId"] == "" {
		return requestObject, errors.New("Could not extract objectId from URI")
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return requestObject, errors.New("Invalid objectid in URI.")
	}
	requestObject.ID = bytesObjectID
	return requestObject, nil
}

func getObjectResponse(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObject,
) error {
	w.Header().Set("Content-Type", "application/json")
	var err error
	var jsonData []byte
	err = nil
	if response.IsDeleted {
		apiResponse := mapping.MapODObjectToDeletedObject(response)
		jsonData, err = json.MarshalIndent(apiResponse, "", "  ")
	} else {
		apiResponse := mapping.MapODObjectToObject(response)
		jsonData, err = json.MarshalIndent(apiResponse, "", "  ")
	}
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
	} else {
		w.Write(jsonData)
	}
	return err
}

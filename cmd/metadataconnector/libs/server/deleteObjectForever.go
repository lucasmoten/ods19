package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) deleteObjectForever(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	var requestObject models.ODObject
	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request in sent format
	requestObject, err = parseDeleteObjectForeverRequest(r)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing JSON")
		return
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObject(requestObject, true)
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

	// If the object is already expunged,
	if dbObject.IsExpunged {
		sendErrorResponse(&w, 410, err, "The referenced object no longer exists.")
		return
	}

	// Call metadata connector to update the object to reflect that it is
	// expunged.  The DAO checks the changeToken and handles the child calls
	dbObject.ModifiedBy = caller.DistinguishedName
	dbObject.ChangeToken = requestObject.ChangeToken
	err = h.DAO.ExpungeObject(dbObject, true)
	if err != nil {
		sendErrorResponse(&w, 500, err, "DAO Error expunging object")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToExpungedObjectResponse(&dbObject)
	deleteObjectForeverResponse(w, r, caller, &apiResponse)
	countOKResponse()
}

func parseDeleteObjectForeverRequest(r *http.Request) (models.ODObject, error) {
	var jsonObject protocol.Object
	var requestObject models.ODObject
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	}

	// Portions from the request URI itself ...
	uri := r.URL.Path
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonObject.ID = uri[matchIndexes[2]:matchIndexes[3]]
			if err != nil {
				return requestObject, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	// Map to internal object type
	requestObject, err = mapping.MapObjectToODObject(&jsonObject)
	return requestObject, err
}

func deleteObjectForeverResponse(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.ExpungedObjectResponse,
) {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

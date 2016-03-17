package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/metadata/models/acm"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) updateObject(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	var requestObject models.ODObject
	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, err, "Could not determine user")
		return
	}

	// Parse Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		requestObject, err = parseUpdateObjectRequestAsJSON(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing JSON")
			return
		}
	default:
		h.sendErrorResponse(w, 501, nil, "Reading from HTML form post not supported")
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object")
		return
	}

	// Check if the user has permissions to update the ODObject
	//		Permission.grantee matches caller, and AllowUpdate is true
	authorizedToUpdate := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName &&
			permission.AllowRead && permission.AllowUpdate {
			authorizedToUpdate = true
			break
		}
	}
	if !authorizedToUpdate {
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// ACM check for whether user has permission to read this object
	// from a clearance perspective
	hasAACAccessToOLDACM, err := h.isUserAllowedForObjectACM(ctx, &dbObject)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error communicating with authorization service")
		return
	}
	if !hasAACAccessToOLDACM {
		h.sendErrorResponse(w, 403, err, "Unauthorized")
		return
	}

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "The object cannot be modified because an ancestor is deleted.")
			return
		case dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
			return
		}
	}

	// Check that assignment as deleted isn't occuring here. Should use deleteObject operations
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		h.sendErrorResponse(w, 428, nil, "Assigning object as deleted through update operation not allowed. Use deleteObject operation")
		return
	}

	// Check that the change token on the object passed in matches the current
	// state of the object in the data store
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) != 0 {
		h.sendErrorResponse(w, 428, nil, "ChangeToken does not match expected value. Object may have been changed by another request.")
		return
	}

	// Check that the parent of the object passed in matches the current state
	// of the object in the data store.
	if bytes.Compare(requestObject.ParentID, dbObject.ParentID) != 0 {
		h.sendErrorResponse(w, 428, nil, "ParentID does not match expected value. Use moveObject to change this objects location.")
		return
	}

	// Check that the owner of the object passed in matches the current state
	// of the object in the data store.
	if len(requestObject.OwnedBy.String) == 0 {
		requestObject.OwnedBy.String = dbObject.OwnedBy.String
		requestObject.OwnedBy.Valid = true
	}
	if strings.Compare(requestObject.OwnedBy.String, dbObject.OwnedBy.String) != 0 {
		h.sendErrorResponse(w, 428, nil, "OwnedBy does not match expected value.  Use changeOwner to transfer ownership.")
		return
	}

	// If ACM provided differs from what is currently set, then need to
	// Check AAC to compare user clearance to NEW metadata Classifications
	// to see if allowed for this user
	if strings.Compare(dbObject.RawAcm.String, requestObject.RawAcm.String) != 0 {
		// Validate ACM
		rawAcmString := requestObject.RawAcm.String
		// Make sure its parseable
		parsedACM, err := acm.NewACMFromRawACM(rawAcmString)
		if err != nil {
			h.sendErrorResponse(w, 428, nil, "ACM provided could not be parsed")
			return
		}
		// Ensure user is allowed this acm
		hasAACAccessToNewACM, err := h.isUserAllowedForObjectACM(ctx, &requestObject)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error communicating with authorization service")
			return
		}
		if !hasAACAccessToNewACM {
			h.sendErrorResponse(w, 403, err, "Unauthorized")
			return
		}
		// Map the parsed acm
		requestObject.ACM = mapping.MapACMToODObjectACM(&parsedACM)
		// Assign existinng database values over top
		// Depends on DAO retrieving the ACM when calling getObject
		requestObject.ACM.ID = dbObject.ACM.ID
		requestObject.ACM.ACMID = dbObject.ACM.ACMID
		requestObject.ACM.ObjectID = dbObject.ACM.ObjectID
		requestObject.ACM.ModifiedBy = caller.DistinguishedName
	}

	// Call metadata connector to update the object in the data store
	// Force the modified by to be that of the caller
	requestObject.ModifiedBy = caller.DistinguishedName
	err = h.DAO.UpdateObject(&requestObject)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "DAO Error updating object")
		return
	}

	// After the update, check that key values have changed...
	if requestObject.ChangeCount <= dbObject.ChangeCount {
		if requestObject.ID == nil {
			log.Println("requestObject.ID = nil")
		}
		if dbObject.ID == nil {
			log.Println("dbObject.ID = nil")
		}
		log.Println(hex.EncodeToString(requestObject.ID))
		log.Println(hex.EncodeToString(dbObject.ID))

		msg := fmt.Sprintf("old changeCount: %d, new changeCount: %d, req id: %s, db id: %s", requestObject.ChangeCount, dbObject.ChangeCount, hex.EncodeToString(requestObject.ID), hex.EncodeToString(dbObject.ID))
		log.Println(msg)
		h.sendErrorResponse(w, 500, nil, "ChangeCount didn't update when processing request "+msg)
		return
	}
	if strings.Compare(requestObject.ChangeToken, dbObject.ChangeToken) == 0 {
		msg := fmt.Sprintf("old token: %s, new token: %s", requestObject.ChangeToken, dbObject.ChangeToken)
		h.sendErrorResponse(w, 500, nil, "ChangeToken didn't update when processing request "+msg)
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODObjectToObject(&requestObject)
	updateObjectResponseAsJSON(w, r, caller, &apiResponse)

}

func parseUpdateObjectRequestAsJSON(r *http.Request) (models.ODObject, error) {
	var jsonObject protocol.Object
	var requestObject models.ODObject
	var err error

	err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	if err != nil {
		return requestObject, err
	}

	// Portions from the request URI itself ...
	uri := r.URL.Path
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)/properties")
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

func updateObjectResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.Object,
) {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

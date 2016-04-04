package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectShare(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		sendErrorResponse(&w, http.StatusInternalServerError, errors.New("Could not determine user"), "Invalid user.")
		return
	}
	_ = caller

	// Parse Request in sent format
	if r.Header.Get("Content-Type") != "application/json" {
		sendErrorResponse(&w, http.StatusBadRequest, errors.New("Bad Request"), "Requires Content-Type: application/json")
		return
	}
	removeObjectShare, err := parseRemoveObjectShareRequest(r)
	if err != nil {
		sendErrorResponse(&w, http.StatusBadRequest, err, "Error parsing JSON")
		return
	}

	// Business Logic...

	// Validate identifiers as decodable to byte value
	requestObject := models.ODObject{}
	requestObject.ID, err = hex.DecodeString(removeObjectShare.ObjectID)
	if err != nil {
		sendErrorResponse(&w, http.StatusBadRequest, err, "Object ID is not a valid format")
		return
	}
	shareID, err := hex.DecodeString(removeObjectShare.ShareID)
	if err != nil {
		sendErrorResponse(&w, http.StatusBadRequest, err, "Share ID is not a valid format")
		return
	}

	// Retrieve existing object from the data store
	dbObject, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		sendErrorResponse(&w, http.StatusInternalServerError, err, "Error retrieving object")
		return
	}

	// Make sure the object isn't deleted. If the object is deleted then cannot make share alterations
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			sendErrorResponse(&w, http.StatusGone, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			sendErrorResponse(&w, http.StatusMethodNotAllowed, err, "The object cannot be modified because an ancestor is deleted.")
			return
		case dbObject.IsDeleted:
			sendErrorResponse(&w, http.StatusGone, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it")
			return
		}
	}

	// Check if the user has permissions to update the ODObject permissions
	//		Permission.grantee matches caller, and AllowShare is true
	// and look for existance of share and changeToken state
	authorizedToShare := false
	shareFound := false
	tokenMatched := false
	var shareToDelete *models.ODObjectPermission
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName &&
			permission.AllowShare {
			authorizedToShare = true
		}
		if bytes.Compare(permission.ID, shareID) == 0 {
			shareFound = true
			shareToDelete = &permission
			if strings.Compare(permission.ChangeToken, removeObjectShare.ChangeToken) == 0 {
				tokenMatched = true
			}
		}
	}
	if !authorizedToShare {
		sendErrorResponse(&w, http.StatusUnauthorized, nil, "Unauthorized")
		return
	}
	if !shareFound {
		sendErrorResponse(&w, http.StatusGone, nil, "Share referenced does not exist")
		return
	}
	if !tokenMatched {
		sendErrorResponse(&w, http.StatusExpectationFailed, nil, "ChangeToken does not match expected value. Share may have been changed by another request.")
		return
	}

	// Update the permission in the database
	shareToDelete.ModifiedBy = caller.DistinguishedName
	shareToDelete.IsDeleted = true
	shareToDelete.DeletedBy.String = caller.DistinguishedName
	dbObjectPermission, err := h.DAO.DeleteObjectPermission(*shareToDelete, removeObjectShare.PropagateToChildren)
	if err != nil {
		sendErrorResponse(&w, http.StatusInternalServerError, err, "Error deleting permission")
		return
	}

	// Response in requested format
	apiResponse := protocol.RemovedObjectShareResponse{}
	apiResponse.DeletedDate = dbObjectPermission.DeletedDate.Time
	removeObjectShareResponse(w, r, caller, &apiResponse)

	countOKResponse()
}

func parseRemoveObjectShareRequest(r *http.Request) (protocol.RemoveObjectShareRequest, error) {
	var jsonObject protocol.RemoveObjectShareRequest
	var err error

	// Depends on this for the changeToken
	err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	if err != nil {
		return jsonObject, err
	}

	// Portions from the request URI itself ...
	uri := r.URL.Path
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)/share/([0-9a-fA-F]*)")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonObject.ObjectID = uri[matchIndexes[2]:matchIndexes[3]]
			if err != nil {
				return jsonObject, errors.New("Object Identifier in Request URI is not a hex string")
			}
			if len(matchIndexes) > 5 {
				jsonObject.ShareID = uri[matchIndexes[4]:matchIndexes[5]]
				if err != nil {
					return jsonObject, errors.New("Share Identifier in Request URI is not a hex string")
				}
			} else {
				return jsonObject, errors.New("Share Identifier was not identified in request path")
			}
		} else {
			return jsonObject, errors.New("Object Identifier was not identified in request path")
		}
	} else {
		return jsonObject, errors.New("Object and Share Identifiers were not identified in request path")
	}

	return jsonObject, err
}

func removeObjectShareResponse(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.RemovedObjectShareResponse,
) {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

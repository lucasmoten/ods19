package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) listObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}
	_ = caller

	targetObject := models.ODObject{}
	var err error

	// Parse Request
	targetObject, err = parseListObjectSharesRequest(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
		return
	}

	// Fetch the matching object
	dbObject, err := h.DAO.GetObject(targetObject, false)
	if err != nil {
		h.sendErrorResponse(w, 404, err, "Resource not found")
	}

	// Check for permission to read this object
	canReadObject := false

	if strings.Compare(dbObject.OwnedBy.String, caller.DistinguishedName) == 0 {
		canReadObject = true
	} else {
		for _, perm := range dbObject.Permissions {
			if perm.AllowRead && perm.AllowShare && perm.Grantee == caller.DistinguishedName {
				canReadObject = true
				break
			}
		}
	}

	if !canReadObject {
		h.sendErrorResponse(w, 403, err, "Insufficient permissions to view shares of this object")
		return
	}
	// Is it deleted?
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "The object cannot be read because an ancestor is deleted.")
			return
		case dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its shares")
			return
		}
	}

	if err != nil {
		log.Println(err)
		h.sendErrorResponse(w, 500, err, "General error")
		return
	}

	// Response in requested format
	listObjectSharesResponseAsJSON(w, mapping.MapODPermissionsToPermissions(&dbObject.Permissions))
	return
}

func parseListObjectSharesRequest(r *http.Request) (models.ODObject, error) {
	requestedObject := models.ODObject{}
	var err error

	// Get object ID from request path
	uri := r.URL.Path
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)/")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			stringObjectID := uri[matchIndexes[2]:matchIndexes[3]]
			byteObjectID, err := hex.DecodeString(stringObjectID)
			if err != nil {
				return requestedObject, errors.New("Object Identifier in Request URI is not a hex string")
			}
			requestedObject.ID = byteObjectID
		}
	}

	// Map to internal object type
	return requestedObject, err
}

func listObjectSharesResponseAsJSON(
	w http.ResponseWriter,
	response []protocol.Permission,
) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

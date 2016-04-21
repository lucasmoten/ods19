package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) listObjectShares(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			sendErrorResponse(&w, 500, errors.New("Could not determine user"), "Invalid user.")
			return
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}

	var err error

	// Parse Request
	captured, _ := CaptureGroupsFromContext(ctx)
	pagingRequest, err := protocol.NewPagingRequest(r, captured, true)
	if err != nil {
		sendErrorResponse(&w, 400, err, "Error parsing request")
		return
	}

	// Fetch the matching object
	targetObject := models.ODObject{}
	targetObject.ID, _ = hex.DecodeString(pagingRequest.ObjectID)
	dbObject, err := h.DAO.GetObject(targetObject, false)
	if err != nil {
		sendErrorResponse(&w, 404, err, "Resource not found")
		return
	}

	// Check for permission to read this object
	canReadObject := false

	if strings.Compare(dbObject.OwnedBy.String, user.DistinguishedName) == 0 {
		canReadObject = true
	} else {
		for _, perm := range dbObject.Permissions {
			if perm.AllowRead && perm.AllowShare && perm.Grantee == user.DistinguishedName {
				canReadObject = true
				break
			}
		}
	}

	if !canReadObject {
		sendErrorResponse(&w, 403, err, "Insufficient permissions to view shares of this object")
		return
	}
	// Is it deleted?
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			sendErrorResponse(&w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			sendErrorResponse(&w, 405, err, "The object cannot be read because an ancestor is deleted.")
			return
		case dbObject.IsDeleted:
			sendErrorResponse(&w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its shares")
			return
		}
	}

	if err != nil {
		log.Println(err)
		sendErrorResponse(&w, 500, err, "General error")
		return
	}

	// Response in requested format
	listObjectSharesResponseAsJSON(w, mapping.MapODPermissionsToPermissions(&dbObject.Permissions))
	countOKResponse()
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

package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) addObjectShare(w http.ResponseWriter, r *http.Request, caller Caller) {
	//Get the json data from the request
	var requestGrant models.ODObjectPermission
	requestGrant, propgateToChildren, err := parseAddObjectShareRequest(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
	}
	log.Printf("Granting:%s to %s", hex.EncodeToString(requestGrant.ObjectID), requestGrant.Grantee)

	// Fetch object to validate
	requestedObject := models.ODObject{}
	requestedObject.ID = requestGrant.ObjectID
	dbObject, err := h.DAO.GetObject(requestedObject, false)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object")
		return
	}

	// Check if the object is deleted
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "Unallowed to share deleted objects.")
			return
		case dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "Use removeObjectFromTrash to restore this object before adding shares.")
			return
		}
	}

	//Get the existing grant, make one for the grantee
	permittedGrant := models.ODObjectPermission{}
	var newGrant models.ODObjectPermission
	for _, permission := range dbObject.Permissions {
		isAllowed :=
			permission.Grantee == caller.DistinguishedName &&
				((dbObject.OwnedBy.String == caller.DistinguishedName) || permission.AllowShare)

		// Add all permissions that apply to the caller to derive overall permitted
		if isAllowed {
			permittedGrant.AllowCreate = permittedGrant.AllowCreate || permission.AllowCreate
			permittedGrant.AllowRead = permittedGrant.AllowRead || permission.AllowRead
			permittedGrant.AllowUpdate = permittedGrant.AllowUpdate || permission.AllowUpdate
			permittedGrant.AllowDelete = permittedGrant.AllowDelete || permission.AllowDelete
			permittedGrant.AllowShare = permittedGrant.AllowShare || permission.AllowShare
			// And capture an encryptKey
			permittedGrant.EncryptKey = make([]byte, 32)
			permittedGrant.EncryptKey = permission.EncryptKey
		}
		// Keep iterating all permissions to build up what is permitted
	}
	if dbObject.TypeName.String != "Folder" {
		if len(permittedGrant.EncryptKey) == 0 {
			log.Printf("Grant was not created")
			h.sendErrorResponse(w, 500, err, "Did not find suitable grant to transfer. EncryptKey not set on permission of non-folder")
			return
		}
		// As a non-folder, encrypt key needs to be applied to grantee.
		// First apply on caller to decrypt
		utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, permittedGrant.EncryptKey)
		//Encrypt to grantee
		utils.ApplyPassphrase(h.MasterKey+requestGrant.Grantee, permittedGrant.EncryptKey)
	}

	// Setup new grant based upon permitted grant permissions
	newGrant.CreatedBy = caller.DistinguishedName
	newGrant.Grantee = requestGrant.Grantee
	// - recalculated encrypt key
	newGrant.EncryptKey = make([]byte, 32)
	newGrant.EncryptKey = permittedGrant.EncryptKey
	// - combined permissions. only allow what is permitted
	newGrant.AllowCreate = permittedGrant.AllowCreate && requestGrant.AllowCreate
	newGrant.AllowRead = permittedGrant.AllowRead && requestGrant.AllowRead
	newGrant.AllowUpdate = permittedGrant.AllowUpdate && requestGrant.AllowUpdate
	newGrant.AllowDelete = permittedGrant.AllowDelete && requestGrant.AllowDelete
	newGrant.AllowShare = permittedGrant.AllowShare && requestGrant.AllowShare
	// - This is an explicit grant
	newGrant.ExplicitShare = true

	// Add to database
	createdPermission, err := h.DAO.AddPermissionToObject(dbObject, &newGrant, propgateToChildren, h.MasterKey)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error updating permission")
		return
	}

	// Response in requested format
	apiResponse := mapping.MapODPermissionToPermission(&createdPermission)
	addObjectShareResponseAsJSON(w, r, caller, &apiResponse)

	//Just signify something that corresponds with the correct http code.
	//User in browser will just know to hit back button.  API user will
	//ignore.
	fmt.Fprintf(w, "ok")
}

func parseAddObjectShareRequest(r *http.Request) (models.ODObjectPermission, bool, error) {
	var requestedGrant protocol.ObjectGrant
	var requestedPermission models.ODObjectPermission
	var err error

	// Parse the JSON body into the requestedGrant
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&requestedGrant)
	if err != nil {
		return requestedPermission, false, errors.New("Unable to decode grant from JSON body")
	}
	// Map to internal permission
	requestedPermission, err = mapping.MapObjectGrantToODPermission(&requestedGrant)
	if err != nil {
		return requestedPermission, false, errors.New("Error mapping grant to permission")
	}

	// Portions from the request URI itself ...
	uri := r.URL.Path
	re, err := regexp.Compile("/object/([0-9a-fA-F]*)/")
	if err != nil {
		return requestedPermission, false, errors.New("Regular Expression for identifing object identifier did not compile")
	}
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			requestedPermission.ObjectID, err = hex.DecodeString(uri[matchIndexes[2]:matchIndexes[3]])
			if err != nil {
				return requestedPermission, false, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	return requestedPermission, requestedGrant.PropogateToChildren, err
}

func addObjectShareResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.Permission,
) {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

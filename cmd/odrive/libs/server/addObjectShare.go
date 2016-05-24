package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

func (h AppServer) addObjectShare(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}

	//Get the json data from the request
	var requestGrant models.ODObjectPermission
	requestGrant, propagateToChildren, err := parseAddObjectShareRequest(r, ctx)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// Normalize the Grantee
	requestGrant.Grantee = config.GetNormalizedDistinguishedName(requestGrant.Grantee)

	hexid := hex.EncodeToString(requestGrant.ObjectID)
	LoggerFromContext(ctx).Info("grant to", zap.String("objectid", hexid), zap.String("grantee", requestGrant.Grantee))
	// Fetch object to validate
	requestedObject := models.ODObject{}
	requestedObject.ID = requestGrant.ObjectID
	dbObject, err := h.DAO.GetObject(requestedObject, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Check if the object is deleted
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return NewAppError(405, err, "Unallowed to share deleted objects.")
		case dbObject.IsDeleted:
			return NewAppError(405, err, "Use removeObjectFromTrash to restore this object before adding shares.")
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
			return NewAppError(500, err, "Did not find suitable grant to transfer. EncryptKey not set on permission of non-folder")
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
	createdPermission, err := h.DAO.AddPermissionToObject(dbObject, &newGrant, propagateToChildren, h.MasterKey)
	if err != nil {
		return NewAppError(500, err, "Error updating permission")
	}

	// Response in requested format
	apiResponse := mapping.MapODPermissionToPermission(&createdPermission)

	// TODO AUDIT Log EventModify
	addObjectShareResponseAsJSON(ctx, w, r, caller, &apiResponse)
	return nil
}

func parseAddObjectShareRequest(r *http.Request, ctx context.Context) (models.ODObjectPermission, bool, error) {
	var requestedGrant protocol.ObjectGrant
	var requestedPermission models.ODObjectPermission
	var err error

	// Parse the JSON body into the requestedGrant
	err = util.FullDecode(r.Body, &requestedGrant)
	if err != nil {
		return requestedPermission, false, errors.New("Unable to decode grant from JSON body")
	}
	// Map to internal permission
	requestedPermission, err = mapping.MapObjectGrantToODPermission(&requestedGrant)
	if err != nil {
		return requestedPermission, false, errors.New("Error mapping grant to permission")
	}

	// Portions from the request URI itself ...
	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return requestedPermission, false, errors.New("Could not get capture groups")
	}
	// Assign requestedPermission with the objectId being shared
	if captured["objectId"] == "" {
		return requestedPermission, false, errors.New("Could not extract objectid from URI")
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return requestedPermission, false, errors.New("Invalid objectid in URI.")
	}
	requestedPermission.ObjectID = bytesObjectID

	// Return it
	return requestedPermission, requestedGrant.PropagateToChildren, err
}

func addObjectShareResponseAsJSON(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.Permission,
) {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		LoggerFromContext(ctx).Error("unable to marshal json", zap.String("err", err.Error()))
		return
	}
	w.Write(jsonData)
}

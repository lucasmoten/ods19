package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	logger := LoggerFromContext(ctx)

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, errors.New("Could not determine user"), "Invalid user.")
	}
	dao := DAOFromContext(ctx)

	// Get the object ID from the request
	bytesObjectID, err := getObjectIDFromContext(ctx)
	if err != nil {
		return NewAppError(400, err, err.Error())
	}

	// Load the existing object
	requestedObject := models.ODObject{}
	requestedObject.ID = bytesObjectID
	dbObject, err := dao.GetObject(requestedObject, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Check Permissions
	allowedToShare, rollupPermission := isUserAllowedToShare(ctx, &dbObject)
	if !allowedToShare {
		return NewAppError(403, errors.New("unauthorized to share"), "Forbidden - User does not have permission to modify shares for an object")
	}

	// Check if the object is deleted
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			return NewAppError(405, err, "Unallowed to set shares for deleted objects.")
		case dbObject.IsDeleted:
			return NewAppError(405, err, "Unallowed to set shares for deleted objects. Use removeObjectFromTrash to restore this object before setting shares.")
		}
	}

	//Get the json data from the request and map to an array of permission objects
	var permissions []models.ODObjectPermission
	var permissionsToAdd []models.ODObjectPermission
	permissions, propagateToChildren, err := parseObjectShareRequest(r, ctx)
	if err != nil {
		return NewAppError(400, err, "Error parsing request")
	}

	// If not granting anything, this is a NOOP and we can return the objects current state
	if len(permissions) == 0 {
		// TODO: Determine if no permissions actually targets 'everyone' ?

		apiResponse := mapping.MapODObjectToObject(&dbObject)
		updateObjectResponseAsJSON(w, r, caller, &apiResponse)
		return nil
	}

	// Iterate the permissions, normalizing the share to derive grantee
	tempObject := models.ODObject{}
	for _, permission := range permissions {

		// Verify that permission settings are allowed for user's rollupPermission
		if !rollupPermission.AllowCreate && permission.AllowCreate {
			return NewAppError(403, fmt.Errorf("User does not have permission to set share with create"), "Forbidden - Unauthorized to set permissions with create")
		}
		if !rollupPermission.AllowRead && permission.AllowRead {
			return NewAppError(403, fmt.Errorf("User does not have permission to set share with read"), "Forbidden - Unauthorized to set permissions with read")
		}
		if !rollupPermission.AllowUpdate && permission.AllowUpdate {
			return NewAppError(403, fmt.Errorf("User does not have permission to set share with update"), "Forbidden - Unauthorized to set permissions with update")
		}
		if !rollupPermission.AllowDelete && permission.AllowDelete {
			return NewAppError(403, fmt.Errorf("User does not have permission to set share with delete"), "Forbidden - Unauthorized to set permissions with delete")
		}
		if !rollupPermission.AllowShare && permission.AllowShare {
			return NewAppError(403, fmt.Errorf("User does not have permission to set share with delegation"), "Forbidden - Unauthorized to set permissions with delegation")
		}

		// re-initialize tempObject ACM from database ACM
		tempObject.RawAcm.Valid = dbObject.RawAcm.Valid
		tempObject.RawAcm.String = dbObject.RawAcm.String

		// Make flattened share for this permission
		herr := h.validateAndFlattenShare(ctx, &permission, &tempObject)
		if herr != nil {
			return herr
		}

		// Get the resultant f_share
		herr, fShareInterface := getACMInterfacePart(&tempObject, "f_share")
		if herr != nil {
			return herr
		}

		// Convert the interface of values into an array
		acmGrants := getStringArrayFromInterface(fShareInterface)
		// Expect only 1 result!
		grantLength := len(acmGrants)
		if grantLength != 1 {
			return NewAppError(500, fmt.Errorf("Resultant f_share from permission after flattening has %d f_share values", grantLength), "Error processing permissions")
		}

		// Assign the f_share value to the grantee
		permission.Grantee = config.GetNormalizedDistinguishedName(acmGrants[0])

		// Assign object id for which this permission is associated
		permission.ObjectID = bytesObjectID

		// Creator
		permission.CreatedBy = caller.DistinguishedName

		// Set as explicit
		permission.ExplicitShare = true

		// --- This is the point where add and remove differ
		// For adding, permissions contains a list of permissions that need to be
		// compared to existing permissions adding records in the database where
		// they dont exist already as well as merge into the acm share and f_share
		// For removing, permissions need to be compared against existing for
		// deleting, and then whatever is left is what needs to be merged as the
		// new acm share

		// Iterate existing permissions on object
		for _, dbPermission := range dbObject.Permissions {
			granteeMatch := strings.Compare(dbPermission.Grantee, permission.Grantee) == 0
			everyoneMatch := strings.Compare(dbPermission.Grantee, models.EveryoneGroup) == 0
			if granteeMatch || everyoneMatch {
				// Discern which permissions this user already has
				if dbPermission.AllowCreate {
					permission.AllowCreate = false
				}
				if dbPermission.AllowRead {
					permission.AllowRead = false
				}
				if dbPermission.AllowUpdate {
					permission.AllowUpdate = false
				}
				if dbPermission.AllowDelete {
					permission.AllowDelete = false
				}
				if dbPermission.AllowShare {
					permission.AllowShare = false
				}
			}
		}

		// Check what is left of missing permissions
		if !permission.AllowCreate && !permission.AllowRead && !permission.AllowUpdate && !permission.AllowDelete && !permission.AllowShare {
			// Nothing left of this permission. so dont add it, continue to next permission
			continue
		}

		// Now we can assign encrypt key, which will set mac based upon permissions being granted
		models.CopyEncryptKey(h.MasterKey, &rollupPermission, &permission)

		// And add it to the list of permissions that will be added
		permissionsToAdd = append(permissionsToAdd, permission)

		// Merge permission.AcmShare into dbObject.RawAcm.String{share}
		if permission.AllowRead {
			herr, sourceInterface := getACMInterfacePart(&dbObject, "share")
			if herr != nil {
				return herr
			}
			interfaceToAdd, err := utils.UnmarshalStringToInterface(permission.AcmShare)
			if err != nil {
				return NewAppError(500, err, "Unable to unmarshal share from permission")
			}
			combinedInterface := CombineInterface(sourceInterface, interfaceToAdd)
			herr = setACMPartFromInterface(ctx, &dbObject, "share", combinedInterface)
			if herr != nil {
				return herr
			}
		}
	}

	// Update the database object now that its ACM has been altered
	dbObject.ModifiedBy = caller.DistinguishedName
	// Reflatten dbObject.RawACM
	err = h.flattenACM(logger, &dbObject)
	if err != nil {
		return NewAppError(500, err, "Error updating permissions when flattening acm")
	}

	// First update the base object that favors ACM change
	err = dao.UpdateObject(&dbObject)
	if err != nil {
		return NewAppError(500, err, "Error updating object")
	}

	// Add these permissions to the database.
	for _, permission := range permissionsToAdd {
		// Add to database
		_, err := dao.AddPermissionToObject(dbObject, &permission, propagateToChildren, h.MasterKey)
		if err != nil {
			return NewAppError(500, err, "Error updating permission on object - add permission")
		}
	}

	// Now fetch updated object
	updatedObject, err := dao.GetObject(requestedObject, false)
	if err != nil {
		return NewAppError(500, err, "Error retrieving object")
	}

	// Return response
	apiResponse := mapping.MapODObjectToObject(&updatedObject)
	updateObjectResponseAsJSON(w, r, caller, &apiResponse)
	return nil
}

func parseObjectShareRequest(r *http.Request, ctx context.Context) ([]models.ODObjectPermission, bool, error) {
	var requestedShare protocol.ObjectShare
	var requestedPermissions []models.ODObjectPermission
	var err error

	// Parse the JSON body into the requestedShare
	err = util.FullDecode(r.Body, &requestedShare)
	if err != nil {
		return requestedPermissions, false, errors.New("Unable to decode share from JSON body")
	}
	// Map to internal permission(s)
	requestedPermissions, err = mapping.MapObjectShareToODPermissions(&requestedShare)
	if err != nil {
		return requestedPermissions, false, errors.New("Error mapping share to permissions")
	}

	// Return it
	return requestedPermissions, requestedShare.PropagateToChildren, nil
}

func getObjectIDFromContext(ctx context.Context) ([]byte, error) {
	var bytesObjectID []byte
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return bytesObjectID, errors.New("Could not get capture groups")
	}
	// Assign requestedPermission with the objectId being shared
	if captured["objectId"] == "" {
		return bytesObjectID, errors.New("Could not extract objectid from URI")
	}
	bytesObjectID, err := hex.DecodeString(captured["objectId"])
	if err != nil {
		return bytesObjectID, errors.New("Invalid objectid in URI.")
	}
	return bytesObjectID, nil
}

func addObjectShareResponseAsJSON(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.Object,
) {
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		LoggerFromContext(ctx).Error("unable to marshal json", zap.String("err", err.Error()))
		return
	}
	w.Write(jsonData)
}

package server

import (
	//"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
)


/* This is used by both createObject and createFolder to do common tasks against created objects
   Returns true if the request is now handled - which happens in the case of errors that terminate
   the http request
 */
func handleCreatePrerequisites(
    h AppServer, 
    requestObject *models.ODObject, 
    requestACM *models.ODACM, 
    w http.ResponseWriter, 
    caller Caller,
) bool {
	// If JavaScript passes parentId as emptry string, set it to nil to satisfy
	// the DAO.
	if string(requestObject.ParentID) == "" {
		requestObject.ParentID = nil
	}

	// Check if parent defined
	if requestObject.ParentID == nil {
		// No parent set, but need to setup permission
		newPermission := models.ODObjectPermission{}
		newPermission.Grantee = caller.DistinguishedName
		newPermission.AllowCreate = true
		newPermission.AllowRead = true
		newPermission.AllowUpdate = true
		newPermission.AllowDelete = true
		requestObject.Permissions = append(requestObject.Permissions, newPermission)
	} else {
		// Parent is defined, retrieve existing parent object from the data store

		parentObject := models.ODObject{}
		parentObject.ID = requestObject.ParentID
		dbParentObject, err := h.DAO.GetObject(parentObject, false)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error retrieving parent object")
			return true
		}

		// Check if the user has permissions to create child objects under the
		// parent.
		//		Permission.grantee matches caller, and AllowCreate is true
		authorizedToCreate := false
		if len(dbParentObject.Permissions) > 0 {
			for _, permission := range dbParentObject.Permissions {
				if permission.Grantee == caller.DistinguishedName &&
					permission.AllowRead && permission.AllowCreate {
					authorizedToCreate = true
					break
				}
			}
		} else {
			log.Println("WARNING: No permissions on the object!")
		}
		if !authorizedToCreate {
			h.sendErrorResponse(w, 403, nil, "Unauthorized")
			return true
		}

		// Make sure the object isn't deleted. To remove an object from the trash,
		// use removeObjectFromTrash call.
		if dbParentObject.IsDeleted {
			switch {
			case dbParentObject.IsExpunged:
				h.sendErrorResponse(w, 410, err, "The object no longer exists.")
				return true
			case dbParentObject.IsAncestorDeleted && !dbParentObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "Unallowed to create child objects under a deleted object.")
				return true
			case dbParentObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "The object under which this object is being created is currently in the trash. Use removeObjectFromTrash to restore it first.")
				return true
			}
		}

		// Copy permissions from parent into request Object
		for _, permission := range dbParentObject.Permissions {
			if !permission.IsDeleted {
				newPermission := models.ODObjectPermission{}
				newPermission.Grantee = permission.Grantee
				isCreator := (permission.Grantee == caller.DistinguishedName)
				newPermission.AllowCreate = permission.AllowCreate || isCreator
				newPermission.AllowRead = permission.AllowRead || isCreator
				newPermission.AllowUpdate = permission.AllowUpdate || isCreator
				newPermission.AllowDelete = permission.AllowDelete || isCreator
				requestObject.Permissions = append(requestObject.Permissions, newPermission)
			}
		}
	}
	log.Printf("There are %d permissions being added..", len(requestObject.Permissions))

	// Disallow creating as deleted
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		h.sendErrorResponse(w, 428, nil, "Creating object in a deleted state is not allowed")
		return true
	}

	// Setup meta data...
	requestObject.CreatedBy = caller.DistinguishedName
	requestObject.OwnedBy.String = caller.DistinguishedName
	requestObject.OwnedBy.Valid = true
	requestACM.CreatedBy = caller.DistinguishedName
    
    return false    
}


// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
) {

	var obj models.ODObject
	var createdObject models.ODObject
	var acm models.ODACM
	var grant models.ODObjectPermission
	var err error

	if r.Method == "POST" {
		grant.Grantee = caller.DistinguishedName
		grant.AllowRead = true
		grant.AllowCreate = true
		grant.AllowUpdate = true
		grant.AllowDelete = true

		obj.CreatedBy = caller.DistinguishedName
		acm.CreatedBy = caller.DistinguishedName

        handleCreatePrerequisites(h, &createdObject, &acm, w, caller)
        
		rName := createRandomName()
		fileKey := createKey()
		iv := createIV()
		obj.ContentConnector.String = rName
		obj.EncryptIV = iv
		grant.EncryptKey = fileKey
		h.acceptObjectUpload(w, r, caller, &obj, &acm, &grant)

		obj.Permissions = make([]models.ODObjectPermission, 1)
		obj.Permissions = append(obj.Permissions, grant)

		createdObject, err = h.DAO.CreateObject(&obj, &acm)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "error storing object")
			return
		}
	}

	//TODO: json response rendering
	w.Header().Set("Content-Type", "application/json")
	protocolObject := mapping.MapODObjectToObject(&createdObject)
	//Write a link back to the user so that it's possible to do an update on this object
	data, err := json.MarshalIndent(protocolObject, "", "  ")
	if err != nil {
		log.Printf("Error marshalling json data:%v", err)
	}
	w.Write(data)
}

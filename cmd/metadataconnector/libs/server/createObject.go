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
<<<<<<< 5d6df11f98da52b9063d690449a20e5353d14fd5
	h AppServer,
	requestObject *models.ODObject,
	requestACM *models.ODACM,
	w http.ResponseWriter,
	caller Caller,
) bool {
=======
    h AppServer, 
    requestObject *models.ODObject, 
    requestACM *models.ODACM, 
    caller Caller,
) (*AppError) {
>>>>>>> Cleaning up error handling
	// If JavaScript passes parentId as emptry string, set it to nil to satisfy
	// the DAO.
	if string(requestObject.ParentID) == "" {
		requestObject.ParentID = nil
	}

	// Check if parent defined
	if requestObject.ParentID == nil {
		// No parent set, but need to setup permission for the creator
		newPermission := models.ODObjectPermission{}
		newPermission.Grantee = caller.DistinguishedName
		newPermission.AllowCreate = true
		newPermission.AllowRead = true
		newPermission.AllowUpdate = true
		newPermission.AllowDelete = true
		newPermission.AllowShare = true
		requestObject.Permissions = append(requestObject.Permissions, newPermission)
	} else {
		// Parent is defined, retrieve existing parent object from the data store

		parentObject := models.ODObject{}
		parentObject.ID = requestObject.ParentID
		dbParentObject, err := h.DAO.GetObject(parentObject, false)
		if err != nil {
			return &AppError{
                Code:500, 
                Err:err, 
                Msg:"Error retrieving parent object",
            }
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
			return &AppError{
                Code:403, 
                Err:nil, 
                Msg:"Unauthorized",
            }
		}

		// Make sure the object isn't deleted. To remove an object from the trash,
		// use removeObjectFromTrash call.
		if dbParentObject.IsDeleted {
			switch {
			case dbParentObject.IsExpunged:
				return &AppError{
                    Code:410, 
                    Err:err, 
                    Msg:"The object no longer exists.",
                }
			case dbParentObject.IsAncestorDeleted && !dbParentObject.IsDeleted:
				return &AppError{
                    Code:405, 
                    Err:err, 
                    Msg:"Unallowed to create child objects under a deleted object.",
                }
			case dbParentObject.IsDeleted:
				return &AppError{
                    Code:405, 
                    Err:err, 
                    Msg:"The object under which this object is being created is currently in the trash. Use removeObjectFromTrash to restore it first.",
                }
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
				newPermission.AllowShare = permission.AllowShare || isCreator
				requestObject.Permissions = append(requestObject.Permissions, newPermission)
			}
		}
	}

	log.Printf("There are %d permissions being added..", len(requestObject.Permissions))

	// Disallow creating as deleted
	if requestObject.IsDeleted || requestObject.IsAncestorDeleted || requestObject.IsExpunged {
		return &AppError{
            Code:428, 
            Err:nil, 
            Msg:"Creating object in a deleted state is not allowed",
        }
	}

	// Setup meta data...
	requestObject.CreatedBy = caller.DistinguishedName
	requestObject.OwnedBy.String = caller.DistinguishedName
	requestObject.OwnedBy.Valid = true
	requestACM.CreatedBy = caller.DistinguishedName
<<<<<<< 5d6df11f98da52b9063d690449a20e5353d14fd5
=======
    
    return nil    
}
>>>>>>> Cleaning up error handling

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
		grant.AllowShare = true

		obj.CreatedBy = caller.DistinguishedName
		acm.CreatedBy = caller.DistinguishedName
        
		rName := createRandomName()
		fileKey := createKey()
		iv := createIV()
		obj.ContentConnector.String = rName
		obj.EncryptIV = iv
		grant.EncryptKey = fileKey
        multipartReader, err := r.MultipartReader()
        if err != nil {
            h.sendErrorResponse(w, 500, err, "Unable to get mime multipart")
            return
        }
		herr, err := h.acceptObjectUpload(multipartReader, caller, &obj, &acm, &grant)
        if herr != nil {
            h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
            return
        }
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

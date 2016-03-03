package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
	"golang.org/x/net/context"

	"decipher.com/oduploader/metadata/models"
)

/**
Almost all code is similar to that of createObject.go, so reuse much code from there.
*/
func (h AppServer) updateObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	var acm models.ODACM //Still blank, but we need to pass it around
	var grant *models.ODObjectPermission

	//Get the object from the database, unedited
	object, err := h.getObjectStreamObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Could not retrieve object")
		return
	}

	if len(object.ID) == 0 {
		h.sendErrorResponse(w, 500, err, "Object for update doesn't have an id")
		return
	}

	//We need a name for the new text, and a new iv
	object.ContentConnector.String = utils.CreateRandomName()
	object.EncryptIV = utils.CreateIV()

	for _, permission := range object.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowUpdate {
			grant = &permission
			break
		}
	}

	if grant == nil {
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	//Descramble key (and rescramble when we go to save object back)
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)
	//Do an upload that is basically the same as for a new object.
	multipartReader, err := r.MultipartReader()
	if err != nil {
		h.sendErrorResponse(w, 400, err, "unable to open multipart reader")
		return
	}
	herr, err := h.acceptObjectUpload(multipartReader, caller, &object, &acm, grant, false)
	if herr != nil {
		h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
		return
	}
	//Rescramble key
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)

	object.ModifiedBy = caller.DistinguishedName
	err = h.DAO.UpdateObject(&object, &acm)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "error storing object")
		return
	}

	object, err = h.getObjectStreamObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Could not retrieve object")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	link := mapping.MapODObjectToObject(&object)
	data, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		log.Printf("Error marshalling json data:%v", err)
		return
	}
	w.Write(data)
}

package server

import (
	//"encoding/hex"
	"encoding/json"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
)

// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
) {

	var obj models.ODObject
	var acm models.ODACM
	var grant models.ODObjectPermission
	var err error
	var parentID string

	if r.Method == "POST" {

		//	grant.ObjectID = obj.ID
		grant.Grantee = caller.DistinguishedName
		grant.AllowRead = true
		grant.AllowCreate = true
		grant.AllowUpdate = true
		grant.AllowDelete = true

		// Set creator
		obj.CreatedBy = caller.DistinguishedName
		acm.CreatedBy = caller.DistinguishedName

		rName := createRandomName()
		fileKey, iv := createKeyIVPair()
		obj.ContentConnector.String = rName
		obj.EncryptIV = iv
		grant.EncryptKey = fileKey
		h.acceptObjectUpload(w, r, caller, &obj, &acm, &grant, &parentID)

		obj.Permissions = make([]models.ODObjectPermission, 1)
		obj.Permissions[0] = grant

		err = h.DAO.CreateObject(&obj, &acm)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "error storing object")
			return
		}
	}

	//TODO: json response rendering
	w.Header().Set("Content-Type", "application/json")
	link := mapping.GetObjectLinkFromObject(config.RootURL, &obj)
	//Write a link back to the user so that it's possible to do an update on this object
	encoder := json.NewEncoder(w)
	encoder.Encode(&link)
}

package server

import (
	"encoding/json"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"log"
)

/**
Almost all code is similar to that of createObject.go, so reuse much code from there.
*/
func (h AppServer) updateObjectStream(w http.ResponseWriter, r *http.Request, caller Caller) {

	var parentID string
	var acm models.ODACM //Still blank, but we need to pass it around
	object, err := h.getObjectStreamObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Could not retrieve object")
	}
	//XXX: Until we actually implement granting, there is exactly one permission per object
	grant := &object.Permissions[0]
	//Descramble key (and rescramble when we go to save object back)
	applyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)

	//We need a name for the new text, and a new iv
	rName := createRandomName()
	_, iv := createKeyIVPair()
	object.ContentConnector.String = rName
	object.EncryptIV = iv

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

	//Do an upload that is basically the same as for a new object.
	h.acceptObjectUpload(w, r, caller, object, &acm, grant, &parentID)
	err = h.DAO.UpdateObject(object, &acm)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "error storing object")
		return
	}
	//Rescramble key
	applyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)

	object, err = h.getObjectStreamObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Could not retrieve object")
	}

	w.Header().Set("Content-Type", "application/json")
	link := mapping.GetObjectLinkFromObject(config.RootURL, object)
	data, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		log.Printf("Error marshalling json data:%v", err)
	}
	w.Write(data)
}

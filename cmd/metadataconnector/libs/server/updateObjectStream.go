package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"encoding/json"
	"net/http"
)

/**
Almost all code is similar to that of createObject.go, so reuse much code from there.
*/
func (h AppServer) updateObjectStream(w http.ResponseWriter, r *http.Request, caller Caller) {

	var acm models.ODACM //Still blank, but we need to pass it around
	object, err := h.getObjectStreamObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Could not retrieve object")
	}
	//XXX: Until we actually implement granting, there is exactly one permission per object
	grant := &object.Permissions[0]
	//Descramble key (and rescramble when we go to save object back)
	applyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)

	rName := createRandomName()
	//_, iv := createKeyIVPair() //This object version has a fresh IV, but same key
	object.ContentConnector.String = rName
	//XXX TODO ... trying to figure out why no decrypt.  Leaving key,iv alone
	//object.EncryptIV = iv

	//Do an upload that is basically the same as for a new object.
	h.acceptObjectUpload(w, r, caller, object, &acm, grant)
	err = dao.UpdateObject(h.MetadataDB, object, &acm)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "error storing object")
		return
	}
	//Rescramble key
	applyPassphrase(h.MasterKey+caller.DistinguishedName, grant.EncryptKey)

	rootURL := "/service/metadataconnector/1.0"

	object, err = h.getObjectStreamObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Could not retrieve object")
	}

	w.Header().Set("Content-Type", "application/json")
	link := GetObjectLinkFromObject(rootURL, object)
	//Write a link back to the user so that it's possible to do an update on this object
	encoder := json.NewEncoder(w)
	encoder.Encode(&link)

}

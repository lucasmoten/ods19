package server

import (
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
)

var rxShare = initRegex("/object/(.*)/share")

// getIDOfObjectTORetrieveStream accepts a passed in URI and finds whether an
// object identifier was passed within it for which the content stream is sought
func getIDOfObjectTORetrieveGrant(uri string) string {
	matchIndexes := rxShare.FindStringSubmatchIndex(uri)
	if len(matchIndexes) == 0 {
		return ""
	}
	value := uri[matchIndexes[2]:matchIndexes[3]]
	return value
}

func (h AppServer) getObjectGrantObject(w http.ResponseWriter, r *http.Request, caller Caller) (*models.ODObject, error) {
	// Identify requested object
	objectID := getIDOfObjectTORetrieveGrant(r.URL.RequestURI())
	// If not valid, return
	if objectID == "" {
		h.sendErrorResponse(w, 400, nil, "URI provided by caller does not specify an object identifier")
		return nil, nil
	}
	// Convert to byte
	objectIDByte, err := hex.DecodeString(objectID)
	if err != nil {
		h.sendErrorResponse(w, 400, nil, "Identifier provided by caller is not a hexidecimal string")
		return nil, err
	}
	// Retrieve from database
	var objectRequested models.ODObject
	objectRequested.ID = objectIDByte
	object, err := h.DAO.GetObject(&objectRequested, false)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cannot get object")
		return nil, err
	}
	return object, nil
}

func (h AppServer) addObjectShare(w http.ResponseWriter, r *http.Request, caller Caller) {
	//Get the json data from the request
	var objectGrant protocol.ObjectGrant
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&objectGrant)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "unable to decode json grant")
		return
	}

	//Get the original object, with respect to the caller.
	object, err := h.getObjectGrantObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "unable to retrieve object to update")
		return
	}
	if object == nil {
		h.sendErrorResponse(w, 500, err, "did not retrieve original object")
		return
	}

	//Get the existing grant, make one for the grantee
	var newGrant models.ODObjectPermission
	for _, permission := range object.Permissions {
		//XXX There is no explicit grant permission
		isAllowed :=
			permission.Grantee == caller.DistinguishedName &&
				permission.AllowRead &&
				permission.AllowUpdate

		if isAllowed {
			newGrant.EncryptKey = make([]byte, 32)
			newGrant.EncryptKey = permission.EncryptKey
			//Decrypt from grantor
			applyPassphrase(h.MasterKey+caller.DistinguishedName, newGrant.EncryptKey)
			//Encrypt to grantee
			applyPassphrase(h.MasterKey+objectGrant.Grantee, newGrant.EncryptKey)
		}
	}
	if len(newGrant.EncryptKey) == 0 {
		log.Printf("Grant was not created")
		h.sendErrorResponse(w, 500, err, "did not find grant to transfer")
	}
	newGrant.Grantee = objectGrant.Grantee
	newGrant.AllowCreate = objectGrant.Create
	newGrant.AllowRead = objectGrant.Read
	newGrant.AllowUpdate = objectGrant.Update
	newGrant.AllowDelete = objectGrant.Delete

	//Now that we have a new grant, we need to add it in
	err = h.DAO.AddPermissionToObject(caller.DistinguishedName, object, &newGrant)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error updating permission")
	}
}

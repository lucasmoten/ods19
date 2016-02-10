package server

import (
	"fmt"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func (h AppServer) updateObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	var requestObject *models.ODObject
	var responseObject models.ODObject

	// Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		requestObject = parseUpdateObjectRequestAsJSON(r)
	default:
		requestObject = parseUpdateObjectRequestAsHTML(r)
	}

	// Logic...
	// 1. dbObject = GetObject using the identity referenced
	dbObject, err := dao.GetObject(h.MetadataDB, requestObject, true)
	if err != nil {

	}
	// 2. Check AAC to compare user clearance to NEW metadata Classifications
	// 		Check if Classification is allowed for this User
	// 3. Check if the user has permissions to update the ODObject
	//		Permission.grantee matches caller, and AllowUpdate is true
	// 4. Does dbObject.changeToken match that of the request object?
	if requestObject.ChangeToken != dbObject.ChangeToken {

	}
	// 5. Call DAO to update the ODObject
	//		Ensure response object has modifieddate, modifiedby, changetoken and
	//		changecount set
	// 6. If permissions are different from dbObject, then need to setup NEW
	//		encrypt keys
	//

	// Response in requested format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		updateObjectResponseAsJSON(w, r, caller, &responseObject)
	default:
		updateObjectResponseAsHTML(w, r, caller, &responseObject)
	}

}

func parseUpdateObjectRequestAsJSON(r *http.Request) *models.ODObject {
	return nil
}
func parseUpdateObjectRequestAsHTML(r *http.Request) *models.ODObject {
	return nil
}

func updateObjectResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObject,
) {
	w.Header().Set("Content-Type", "application/json")

}

func updateObjectResponseAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObject,
) {

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "updateObject", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)

}

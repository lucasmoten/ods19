package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func (h AppServer) updateObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	var requestObject *models.ODObject
	var err error

	// Parse Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		requestObject, err = parseUpdateObjectRequestAsJSON(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing JSON")
			return
		}
	default:
		h.sendErrorResponse(w, 501, nil, "Reading from HTML form post not supported")
		requestObject = parseUpdateObjectRequestAsHTML(r)
	}

	// Business Logic...

	// 1. dbObject = GetObject using the identity referenced
	dbObject, err := dao.GetObject(h.MetadataDB, requestObject, true)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object")
		return
	}

	// TODO
	// 2. Check AAC to compare user clearance to NEW metadata Classifications
	// 		Check if Classification is allowed for this User

	// 3. Check if the user has permissions to update the ODObject
	//		Permission.grantee matches caller, and AllowUpdate is true
	authorizedToUpdate := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowUpdate {
			authorizedToUpdate = true
		}
	}
	if !authorizedToUpdate {
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// 4. Does dbObject.changeToken match that of the request object?
	if requestObject.ChangeToken != dbObject.ChangeToken {
		h.sendErrorResponse(w, 428, nil, "ChangeToken does not match expected value. Object may have been changed by another request.")
		return
	}

	// 5. Call DAO to update the ODObject
	// TODO: Handle ACM. This needs to be parsed as separate object? Maybe the
	// model should have it nested like has been changed in the wiki page
	requestObject.ModifiedBy = caller.DistinguishedName
	dao.UpdateObject(h.MetadataDB, requestObject, nil)
	if requestObject.ChangeCount <= dbObject.ChangeCount {
		h.sendErrorResponse(w, 500, nil, "ChangeCount didn't update when processing request")
		return
	}
	if requestObject.ChangeToken == dbObject.ChangeToken {
		h.sendErrorResponse(w, 500, nil, "ChangeToken didn't update when processing request")
		return
	}

	// 6. If permissions are different from dbObject, then need to setup NEW
	//		encrypt keys
	// TODO: There is a similar todo in updateobject dao, and its unclear at this
	// point whether it should be done in the dao, or outside here in the bizlogic
	// since that may need to also update the content stream with new EncryptKey

	// Response in requested format
	switch {
	case r.Header.Get("Content-Type") == "multipart/form-data":
		fallthrough
	case r.Header.Get("Content-Type") == "application/json":
		updateObjectResponseAsJSON(w, r, caller, requestObject)
	default:
		updateObjectResponseAsHTML(w, r, caller, requestObject)
	}

}

func parseUpdateObjectRequestAsJSON(r *http.Request) (*models.ODObject, error) {
	var jsonObject models.ODObject
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonObject)
	case r.Header.Get("Content-Type") == "multipart/form-data":
		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			return nil, err
		}
		for {
			part, err := multipartReader.NextPart()
			if err != nil {
				return nil, err
			}
			switch {
			case part.Header.Get("Content-Type") == "application/json":

				// Read in the JSON - up to 10K
				valueAsBytes := make([]byte, 10240)
				n, err := part.Read(valueAsBytes)
				if err != nil {
					return nil, err
				}
				err = (json.NewDecoder(bytes.NewReader(valueAsBytes[0:n]))).Decode(&jsonObject)
			case part.Header.Get("Content-Disposition") == "form-data":
				// TODO: Maybe these header checks need to be if the value begins with?
			}
		}
	}

	return &jsonObject, err
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
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

func updateObjectResponseAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObject,
) {

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "updateObject", caller.DistinguishedName)

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)

	fmt.Fprintf(w, pageTemplateEnd)

}

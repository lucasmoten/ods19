package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func (h AppServer) getObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	var requestObject *models.ODObject
	var err error

	// Parse Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		requestObject, err = parseGetObjectRequestAsJSON(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing JSON")
			return
		}
	default:
		requestObject, err = parseGetObjectRequestAsHTML(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing URI")
			return
		}
	}

	// Business Logic...

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(h.MetadataDB, requestObject, true)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Error retrieving object")
		return
	}

	// Check if the user has permissions to read the ODObject
	//		Permission.grantee matches caller, and AllowRead is true
	authorizedToRead := false
	for _, permission := range dbObject.Permissions {
		if permission.Grantee == caller.DistinguishedName && permission.AllowRead {
			authorizedToRead = true
		}
	}
	if !authorizedToRead {
		h.sendErrorResponse(w, 403, nil, "Unauthorized")
		return
	}

	// TODO
	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User

	// Make sure the object isn't deleted. To remove an object from the trash,
	// use removeObjectFromTrash call.
	if dbObject.IsDeleted {
		switch {
		case dbObject.IsExpunged:
			h.sendErrorResponse(w, 410, err, "The object no longer exists.")
			return
		case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
			h.sendErrorResponse(w, 405, err, "The object cannot be retreived because an ancestor is deleted.")
			return
		}
	}

	// Response in requested format
	switch {
	case r.Header.Get("Content-Type") == "multipart/form-data":
		fallthrough
	case r.Header.Get("Content-Type") == "application/json":
		getObjectResponseAsJSON(w, r, caller, dbObject)
	default:
		getObjectResponseAsHTML(w, r, caller, dbObject)
	}

}

func parseGetObjectRequestAsJSON(r *http.Request) (*models.ODObject, error) {
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

	// Portions from the request URI itself ...
	uri := r.URL.RequestURI()
	re, _ := regexp.Compile("/object/(.*)/properties")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonObject.ID, err = hex.DecodeString(uri[matchIndexes[2]:matchIndexes[3]])
			if err != nil {
				return nil, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	return &jsonObject, err
}
func parseGetObjectRequestAsHTML(r *http.Request) (*models.ODObject, error) {
	var htmlObject models.ODObject
	var err error

	// Portions from the request URI itself ...
	uri := r.URL.RequestURI()
	re, _ := regexp.Compile("/object/(.*)/properties")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			htmlObject.ID, err = hex.DecodeString(uri[matchIndexes[2]:matchIndexes[3]])
			if err != nil {
				return nil, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	return &htmlObject, err
}

func getObjectResponseAsJSON(
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

func getObjectResponseAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObject,
) {

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "getObject", caller.DistinguishedName)

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)

	fmt.Fprintf(w, pageTemplateEnd)

}

package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
)

func (h AppServer) getObjectStream(w http.ResponseWriter, r *http.Request, caller Caller) {

	// Identify requested object
	objectID := getIDOfObjectTORetrieveStream(r.URL.RequestURI())
	// If not valid, return
	if objectID == "" {
		h.sendErrorResponse(w, 400, nil, "URI provided by caller does not specify an object identifier")
		return
	}
	// Convert to byte
	objectIDByte, err := hex.DecodeString(objectID)
	if err != nil {
		h.sendErrorResponse(w, 400, nil, "Identifier provided by caller is not a hexidecimal string")
		return
	}
	// Retrieve from database
	var objectRequested models.ODObject
	objectRequested.ID = objectIDByte
	object, err := dao.GetObject(h.MetadataDB, &objectRequested, false)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cannot get object")
		return
	}

	// Authorization checks
	canRetrieve := false
	if object.OwnedBy.String == caller.DistinguishedName {
		canRetrieve = true
	}
	// TODO Check object permission grants

	if !canRetrieve {
		h.sendErrorResponse(w, 403, nil, "Caller does not have permission to the requested object")
	}

	// TODO: Based upon object metadata, get the object from S3
	//		object.ContentConnector
	//		object.ContentHash

	hasStream := false
	if !hasStream {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, pageTemplateStart, "getObjectStream", caller.DistinguishedName)
		fmt.Fprintf(w, "No content")
		fmt.Fprintf(w, pageTemplateEnd)
		return
	}

	contentType := "text/html"
	if object.ContentType.Valid {
		contentType = object.ContentType.String
	}
	w.Header().Set("Content-Type", contentType)
	if object.ContentSize.Valid {
		w.Header().Set("Content-Length", strconv.FormatInt(object.ContentSize.Int64, 10))
	}
	fmt.Fprintf(w, pageTemplateStart, "getObjectStream", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}

// getIDOfObjectTORetrieveStream accepts a passed in URI and finds whether an
// object identifier was passed within it for which the content stream is sought
func getIDOfObjectTORetrieveStream(uri string) string {
	re, _ := regexp.Compile("/object/(.*)/stream")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) == 0 {
		return ""
	}
	value := uri[matchIndexes[2]:matchIndexes[3]]
	return value
}

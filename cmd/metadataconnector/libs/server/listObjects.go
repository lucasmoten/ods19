package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

type listObjectsRequest struct {
	pageNumber int
	pageSize   int
}

func (h AppServer) listObjects(w http.ResponseWriter, r *http.Request) {
	//peerDistinguishedName := config.GetDistinguishedName(r.TLS.PeerCertificates[0])
	//userDistinguishedName := r.Header.Get("USER_DN")
	//externalSystemDistinguishedName := r.Header.Get("EXTERNAL_SYS_DN")
	who := config.GetDistinguishedName(r.TLS.PeerCertificates[0])

	// Find parentId from request URI
	parentID := getParentIDToListObjects(r)

	// Find pageNmber and pageSize from the body
	pageNumber := 1
	pageSize := 20

	// Initialize output
	switch {
	case r.Method == "GET":
		// Output
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, pageTemplateStart, "listObjects", who)
		fmt.Fprintf(w, pageTemplatePager, "listObjectsPager")
		fmt.Fprintf(w, pageTemplateDataTable, "listObjectsResults")
		fmt.Fprintf(w, pageTemplateEnd)
	case r.Method == "POST":
		jsonRequest := getListObjectsRequestAsJSON(r)
		pageNumber = jsonRequest.pageNumber
		pageSize = jsonRequest.pageSize
	}

	// Fetch the matching objects
	var response models.ODObjectResultset
	var err error
	if parentID != "" {
		response, err = dao.GetChildObjectsWithPropertiesByOwner(h.MetadataDB, "createddate desc", pageNumber, pageSize, parentID, who)
	} else {
		response, err = dao.GetRootObjectsWithPropertiesByOwner(h.MetadataDB, "createddate desc", pageNumber, pageSize, who)
	}
	if err != nil {
		panic(err.Error())
	}
	// Get objects from response
	objects := response.Objects

	fmt.Fprintf(w, "Page "+strconv.Itoa(response.PageNumber)+" of "+strconv.Itoa(response.PageCount)+".<br />")
	fmt.Fprintf(w, "Page Size: "+strconv.Itoa(response.PageSize)+", Page Rows: "+strconv.Itoa(response.PageRows)+", Total Rows: "+strconv.Itoa(response.TotalRows)+"<br />")
	fmt.Fprintf(w, `<table id="listObjectsResults">`)
	fmt.Fprintf(w, `<tr><td>Name</td><td>Created Date</td><td>Created By</td></tr>`)
	for idx := range objects {
		object := objects[idx]

		fmt.Fprintf(w, "<tr><td>")
		fmt.Fprintf(w, "<a href='/object/")
		fmt.Fprintf(w, hex.EncodeToString(object.ID))
		fmt.Fprintf(w, "/list'>")
		fmt.Fprintf(w, object.Name)
		fmt.Fprintf(w, "</a>")
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, getFormattedDate(object.CreatedDate))
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, object.CreatedBy)
		fmt.Fprintf(w, "</td></tr>")
	}
	fmt.Fprintf(w, "</table>")

}

func getFormattedDate(t time.Time) string {
	return t.Format(time.RFC3339)
}

func getParentIDToListObjects(r *http.Request) string {
	/*
		RESTful API Request

		GET /services/object-drive/object/{objectId}/list HTTP/1.1
		Host: fully.qualified.domain.name
		Content-Type: application/json;
		Content-Length: nnn

		{
		  "pageNumber": "{pageNumber}",
		  "pageSize": {pageSize}
		}
	*/
	re, _ := regexp.Compile("/object/(.*)/list")
	uri := r.URL.RequestURI()
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) == 0 {
		return ""
	}
	value := uri[matchIndexes[2]:matchIndexes[3]]
	return value
}

func getListObjectsRequestAsJSON(r *http.Request) listObjectsRequest {
	decoder := json.NewDecoder(r.Body)
	var result listObjectsRequest
	err := decoder.Decode(&result)
	if err != nil {
		//TODO: Log it

		// Force to page 1, size of 20
		result.pageNumber = 1
		result.pageSize = 20
	}
	if result.pageNumber < 1 {
		result.pageNumber = 1
	}
	if result.pageSize < 1 {
		result.pageSize = 20
	}
	return result
}

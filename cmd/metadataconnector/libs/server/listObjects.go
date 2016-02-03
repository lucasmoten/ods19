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

// listObjects is a method handler on AppServer for implementing the listObjects
// microservice operation.  If an ID is given in the request URI, then it is
// used to list the children within it, otherwise, the root for the given user
// is listed.  For a user, the root is defined as those objects that they own
// which have no parent identifier set.
// Request format:
//				GET /services/object-drive/object/{objectId}/list HTTP/1.1
//				Host: fully.qualified.domain.name
//				Content-Type: application/json;
//				Content-Length: nnn
//
//				{
//					"pageNumber": "{pageNumber}",
//					"pageSize": {pageSize}
//				}
// TODO: Implement proper paging and and result information
// TODO: Convert response to JSON
func (h AppServer) listObjects(w http.ResponseWriter, r *http.Request, caller Caller) {
	rootURL := "/service/metadataconnector/1.0"
	// Find parentId from request URI
	parentID := getParentIDToListObjects(r.URL.RequestURI())

	// Find pageNmber and pageSize from the body
	pageNumber := 1
	pageSize := 20

	// Initialize output
	switch {
	case r.Method == "GET":
		// Output
	case r.Method == "POST":
		jsonRequest := getListObjectsRequestAsJSON(r)
		pageNumber = jsonRequest.pageNumber
		pageSize = jsonRequest.pageSize
	}

	// Fetch the matching objects
	linkToParent := ""
	var response models.ODObjectResultset
	var err error
	if parentID != "" {
		var parentObject models.ODObject
		parentObject.ID, err = hex.DecodeString(parentID)
		if err != nil {
			w.WriteHeader(400)
			fmt.Println("Parent Identifier provided by caller is not a hexidecimal string")
			return
		}
		response, err = dao.GetChildObjectsWithPropertiesByOwner(h.MetadataDB, "createddate desc", pageNumber, pageSize, &parentObject, caller.DistinguishedName)
		loadedParent, err := dao.GetObject(h.MetadataDB, &parentObject, false)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Unable to retrieve object represented by Parent Identifier: %s", err)
		}
		if len(loadedParent.ParentID) > 0 {
			linkToParent = fmt.Sprintf("<a href='%s/object/%s/list'>Up to Parent</a><br />", rootURL, hex.EncodeToString(loadedParent.ParentID))

		} else {
			linkToParent = fmt.Sprintf("<a href='%s/objects'>Up to Root</a><br />", rootURL)
		}
	} else {
		response, err = dao.GetRootObjectsWithPropertiesByOwner(h.MetadataDB, "createddate desc", pageNumber, pageSize, caller.DistinguishedName)
	}
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error: %s", err)
		return
	}
	// Get objects from response
	objects := response.Objects

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "listObjects", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplatePager, "listObjectsPager")
	fmt.Fprintf(w, pageTemplateDataTable, "listObjectsResults")
	fmt.Fprintf(w, pageTemplateEnd)
	fmt.Fprintf(w, linkToParent)
	fmt.Fprintf(w, "Page "+strconv.Itoa(response.PageNumber)+" of "+strconv.Itoa(response.PageCount)+".<br />")
	fmt.Fprintf(w, "Page Size: "+strconv.Itoa(response.PageSize)+", Page Rows: "+strconv.Itoa(response.PageRows)+", Total Rows: "+strconv.Itoa(response.TotalRows)+"<br />")
	fmt.Fprintf(w, `<table id="listObjectsResults">`)
	fmt.Fprintf(w, `<tr><td>Name</td><td>Type</td><td>Created Date</td><td>Created By</td></tr>`)
	rootURL := "/service/metadataconnector/1.0"
	for idx := range objects {
		object := objects[idx]

		fmt.Fprintf(w, "<tr><td>")
		switch {
		case object.TypeName.String == "Folder":
			fmt.Fprintf(w, "<a href='%s/object/", rootURL)
			fmt.Fprintf(w, hex.EncodeToString(object.ID))
			fmt.Fprintf(w, "/list'>")
			fmt.Fprintf(w, object.Name)
			fmt.Fprintf(w, "</a>")
		default:
			fmt.Fprintf(w, "<a href='%s/object/", rootURL)
			fmt.Fprintf(w, hex.EncodeToString(object.ID))
			fmt.Fprintf(w, "/stream'>")
			fmt.Fprintf(w, object.Name)
			fmt.Fprintf(w, "</a>")
		}
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, object.TypeName.String)
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, getFormattedDate(object.CreatedDate))
		fmt.Fprintf(w, "</td><td>")
		fmt.Fprintf(w, config.GetCommonName(object.CreatedBy))
		fmt.Fprintf(w, "</td></tr>")
	}
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, `
	<hr/>
	<form method="post" action="%s/folder" enctype="multipart/form-data">
	<input type="hidden" name="parentId" value="%s" />
	<input type="hidden" name="type" value="Folder" />
	<table>
		<tr>
			<td>New Folder Name</td>
			<td><input type="text" id="title" name="title" /></td>
			<td><input type="submit" value="Create" /></td>
		</tr>
	</table>
	</form>
			`, rootURL, parentID)
}

// getFormattedDate formats a passed in time as RFC3339 format, which is
// basically:    YYYY-MM-DDTHH:mm:ss.nnnZ
// TODO: Move this utility method to a common file to make it clear its
// available by all operations
func getFormattedDate(t time.Time) string {
	return t.Format(time.RFC3339)
}

// getParentIDToListObjects accepts a passed in URI and finds whether an
// object identifier was passed within it for which child items are sought.
func getParentIDToListObjects(uri string) string {
	re, _ := regexp.Compile("/object/(.*)/list")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) == 0 {
		return ""
	}
	value := uri[matchIndexes[2]:matchIndexes[3]]
	return value
}

// getListObjectsRequestAsJSON is used for parsing the request as json to get
// the pageNumber and pageSize of results requested.
// TODO: This especially needs a test as it is as yet unvalidated
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

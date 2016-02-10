package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"encoding/hex"
	"encoding/json"
	"fmt"
	//"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type listObjectsRequest struct {
	pageNumber int // `json:"pageNumber"`
	pageSize   int // `json:"pageSize"`
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

	if r.Header.Get("Content-Type") == "application/json" {
		// do RESTful request
	} else {
		// render template with submit button
		tmpl := h.TemplateCache.Lookup("listObjects.html")
		data := struct{ ParentID string }{parentID}
		tmpl.Execute(w, data)
	}

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
			h.sendErrorResponse(w, 400, err, "ParentID provided by caller is not a hex string")
			return
		}
		response, err = dao.GetChildObjectsWithPropertiesByOwner(
			h.MetadataDB, "createddate desc", pageNumber, pageSize, &parentObject, caller.DistinguishedName)
		loadedParent, err := dao.GetObject(h.MetadataDB, &parentObject, false)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Unable to retrieve ParentID")
		}
		if len(loadedParent.ParentID) > 0 {
			linkToParent = fmt.Sprintf("<a href='%s/object/%s/list'>Up to Parent</a><br />", rootURL, hex.EncodeToString(loadedParent.ParentID))

		} else {
			linkToParent = fmt.Sprintf("<a href='%s/objects'>Up to Root</a><br />", rootURL)
		}
	} else {
		linkToParent = ""
		response, err = dao.GetRootObjectsWithPropertiesByOwner(h.MetadataDB, "createddate desc", pageNumber, pageSize, caller.DistinguishedName)
	}
	if err != nil {
		h.sendErrorResponse(w, 500, err, "General error")
		return
	}
	if r.Header.Get("Content-Type") == "application/json" {
		h.listObjectsAsJSON(w, r, caller, &response, parentID, linkToParent, rootURL)
	} else {
		h.listObjectsAsHTML(w, r, caller, &response, parentID, linkToParent, rootURL)
	}
}

// ObjectLinkResponse is the container for returned data
type ObjectLinkResponse struct {
	TotalRows  int
	PageCount  int
	PageNumber int
	PageSize   int
	PageRows   int
	Objects    []ObjectLink
}

// ObjectLink is the links as exposed to the user of the API
type ObjectLink struct {
	URL        string
	Name       string
	Type       string
	CreateDate string
	CreatedBy  string
	Size       int64
	ACM        string
}

func (h AppServer) listObjectsAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObjectResultset,
	parentID string,
	linkToParent string,
	rootURL string,
) {
	w.Header().Set("Content-Type", "application/json")
	var links []ObjectLink
	for idx := range response.Objects {
		object := response.Objects[idx]
		link := ObjectLink{
			URL:        rootURL + "/object/" + hex.EncodeToString(object.ID),
			Name:       object.Name,
			Type:       object.TypeName.String,
			CreateDate: getFormattedDate(object.CreatedDate),
			CreatedBy:  config.GetCommonName(object.CreatedBy),
			Size:       object.ContentSize.Int64,
			ACM:        object.RawAcm.String,
		}
		links = append(links, link)
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(links)
}

func (h AppServer) listObjectsAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObjectResultset,
	parentID string,
	linkToParent string,
	rootURL string,
) {
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
	fmt.Fprintf(w, `<tr><td>Name</td><td>Type</td><td>Created Date</td><td>Created By</td><td>Size</td><td>ACM</td></tr>`)
	for idx := range objects {
		object := objects[idx]
		fmt.Fprintf(w, "<tr>")
		switch {
		case object.TypeName.String == "Folder":
			fmt.Fprintf(
				w,
				"<td><a href='%s/object/%s/list'>%s</a></td>",
				rootURL,
				hex.EncodeToString(object.ID),
				object.Name,
			)
		default:
			fmt.Fprintf(
				w,
				"<td><a href='%s/object/%s/stream'>%s</a></td>",
				rootURL,
				hex.EncodeToString(object.ID),
				object.Name,
			)
		}

		fmt.Fprintf(w, "<td>%s</td>", object.TypeName.String)
		fmt.Fprintf(w, "<td>%s</td>", getFormattedDate(object.CreatedDate))
		fmt.Fprintf(w, "<td>%s</td>", config.GetCommonName(object.CreatedBy))
		fmt.Fprintf(w, "<td>%d</td>", object.ContentSize.Int64)
		fmt.Fprintf(w, "<td>%s</td>", object.RawAcm.String)
		fmt.Fprintf(w, "</tr>")
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

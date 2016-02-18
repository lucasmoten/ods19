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
	"strconv"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
)

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

	var parentObject *models.ODObject
	var pagingRequest *protocol.PagingRequest
	var err error

	// Parse Request in sent format
	switch {
	case r.Header.Get("Content-Type") == "application/json":
		parentObject, pagingRequest, err = parseListObjectsRequestAsJSON(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing JSON")
			return
		}
	default:
		parentObject, pagingRequest, err = parseListObjectsRequestAsHTML(r)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "Error parsing HTML request")
			return
		}
	}

	// Fetch the matching objects
	var response models.ODObjectResultset
	if parentObject.ID == nil {
		// Requesting root
		response, err = h.DAO.GetRootObjectsWithPropertiesByOwner(
			"createddate desc",
			pagingRequest.PageNumber,
			pagingRequest.PageSize,
			caller.DistinguishedName,
		)
	} else {
		// Requesting children of an object. Load it...
		dbObject, err := h.DAO.GetObject(parentObject, false)
		if err != nil {
			log.Println(err)
			h.sendErrorResponse(w, 500, err, "Error retrieving object")
			return
		}
		// Check for permission to read this object
		canReadObject := false
		for _, perm := range dbObject.Permissions {
			if perm.AllowRead && perm.Grantee == caller.DistinguishedName {
				canReadObject = true
				break
			}
		}
		if !canReadObject {
			h.sendErrorResponse(w, 403, err, "Insufficient permissions to list contents of this object")
			return
		}
		// Is it deleted?
		if dbObject.IsDeleted {
			switch {
			case dbObject.IsExpunged:
				h.sendErrorResponse(w, 410, err, "The object no longer exists.")
				return
			case dbObject.IsAncestorDeleted && !dbObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "The object cannot be read because an ancestor is deleted.")
				return
			case dbObject.IsDeleted:
				h.sendErrorResponse(w, 405, err, "The object is currently in the trash. Use removeObjectFromTrash to restore it before listing its contents")
				return
			}
		}

		// Get the objects
		response, err = h.DAO.GetChildObjectsWithPropertiesByOwner(
			"createddate desc",
			pagingRequest.PageNumber,
			pagingRequest.PageSize,
			parentObject,
			caller.DistinguishedName,
		)
	}
	if err != nil {
		log.Println(err)
		h.sendErrorResponse(w, 500, err, "General error")
		return
	}

	// Response in requested format
	switch {
	case r.Header.Get("Content-Type") == "multipart/form-data":
		fallthrough
	case r.Header.Get("Content-Type") == "application/json":
		apiResponse := mapping.MapODObjectResultsetToObjectResultset(&response)
		listObjectsResponseAsJSON(w, r, caller, &apiResponse)
	default:
		listObjectsResponseAsHTML(w, r, caller, h.DAO, parentObject, &response)
	}
	return
}

func parseListObjectsRequestAsHTML(r *http.Request) (*models.ODObject, *protocol.PagingRequest, error) {
	var object models.ODObject
	var paging protocol.PagingRequest
	var err error
	// defaults
	paging.PageNumber = 1
	paging.PageSize = 20

	// Portions from the request URI itself ...
	uri := r.URL.RequestURI()
	re, _ := regexp.Compile("/object/(.*)/list")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			object.ID, err = hex.DecodeString(uri[matchIndexes[2]:matchIndexes[3]])
			if err != nil {
				return nil, nil, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	// Query string arguments
	pageNumber := r.URL.Query().Get("PageNumber")
	if len(pageNumber) > 0 {
		paging.PageNumber, err = strconv.Atoi(pageNumber)
		if err != nil {
			// TODO: Log this parsing error
			err = nil
			// default
			paging.PageNumber = 1
		}
	}
	pageSize := r.URL.Query().Get("PageSize")
	if len(pageSize) > 0 {
		paging.PageSize, err = strconv.Atoi(pageSize)
		if err != nil {
			// TODO: Log this parsing Error
			err = nil
			// default
			paging.PageSize = 20
		}
	}

	return &object, &paging, err
}
func parseListObjectsRequestAsJSON(r *http.Request) (*models.ODObject, *protocol.PagingRequest, error) {
	var jsonObject protocol.Object
	var jsonPaging protocol.PagingRequest
	jsonPaging.PageNumber = 1
	jsonPaging.PageSize = 20
	var err error

	switch {
	case r.Header.Get("Content-Type") == "application/json":
		err = (json.NewDecoder(r.Body)).Decode(&jsonPaging)
		if err != nil {
			// TODO: log it?
			// but this is paging, so goto defaults and reset the Error
			jsonPaging.PageNumber = 1
			jsonPaging.PageSize = 20
			err = nil
		}
	case r.Header.Get("Content-Type") == "multipart/form-data":
		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			// TODO: log it?
			// but this is paging, so goto defaults and reset the Error
			jsonPaging.PageNumber = 1
			jsonPaging.PageSize = 20
			err = nil
		} else {
			for {
				part, err := multipartReader.NextPart()
				if err != nil {
					// TODO: log it?
					// but this is paging, so goto defaults and reset the Error
					jsonPaging.PageNumber = 1
					jsonPaging.PageSize = 20
					err = nil
				} else {
					switch {
					case part.Header.Get("Content-Type") == "application/json":

						// Read in the JSON - up to 10K
						valueAsBytes := make([]byte, 10240)
						n, err := part.Read(valueAsBytes)
						if err != nil {
							// TODO: log it?
							// but this is paging, so goto defaults and reset the Error
							jsonPaging.PageNumber = 1
							jsonPaging.PageSize = 20
							err = nil
						} else {
							err = (json.NewDecoder(bytes.NewReader(valueAsBytes[0:n]))).Decode(&jsonPaging)
							if err != nil {
								// TODO: log it?
								// but this is paging, so goto defaults and reset the Error
								jsonPaging.PageNumber = 1
								jsonPaging.PageSize = 20
								err = nil
							}
						}
					case part.Header.Get("Content-Disposition") == "form-data":
						// TODO: Maybe these header checks need to be if the value begins with?
					}
				}
			}
		}
	}

	// Portions from the request URI itself ...
	uri := r.URL.RequestURI()
	re, _ := regexp.Compile("/object/(.*)/list")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) != 0 {
		if len(matchIndexes) > 3 {
			jsonObject.ID, err = hex.DecodeString(uri[matchIndexes[2]:matchIndexes[3]])
			if err != nil {
				return nil, nil, errors.New("Object Identifier in Request URI is not a hex string")
			}
		}
	}

	// Map to internal object type
	object := mapping.MapObjectToODObject(&jsonObject)
	return &object, &jsonPaging, err
}

func listObjectsResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.ObjectResultset,
) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}

func listObjectsResponseAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	dao dao.DataAccessLayer,
	parentObject *models.ODObject,
	response *models.ODObjectResultset,
) {
	canCreateFolder := false
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "listObjects", caller.DistinguishedName)
	// Vertical Navigation (Up to Parent)
	// Check if the object referenced is the root for displaying a link up
	if parentObject.ID != nil {
		// Not the root, get the parent of the referenced Object
		dbObject, err := dao.GetObject(parentObject, false)
		linkToParent := ""
		if err != nil {
			// swallow this error for HTML output
		} else {
			if len(dbObject.ParentID) > 0 {
				linkToParent = fmt.Sprintf("<a href='%s/object/%s/list'>Up to Parent</a><br />", config.RootURL, hex.EncodeToString(dbObject.ParentID))

			} else {
				linkToParent = fmt.Sprintf("<a href='%s/objects'>Up to Root</a><br />", config.RootURL)
			}
		}
		fmt.Fprintf(w, linkToParent)
		// Check permission to create folder, for displaying form later
		for _, perm := range dbObject.Permissions {
			if perm.AllowCreate && perm.Grantee == caller.DistinguishedName {
				canCreateFolder = true
				break
			}
		}
	} else {
		canCreateFolder = true
	}
	// Horizontal Navigation (pages)
	fmt.Fprintf(w, "Page "+strconv.Itoa(response.PageNumber)+" of "+strconv.Itoa(response.PageCount)+".<br />")
	fmt.Fprintf(w, "Page Size: "+strconv.Itoa(response.PageSize)+", Page Rows: "+strconv.Itoa(response.PageRows)+", Total Rows: "+strconv.Itoa(response.TotalRows)+"<br />")
	if response.PageCount > 1 {
		fmt.Fprintf(w, createPagerAsHTML(r.URL.Path, response.PageCount, response.PageNumber, response.PageSize))
	}
	// Now render the table of objects...
	fmt.Fprintf(w, `<table id="listObjectsResults">`)
	fmt.Fprintf(w, `<tr><td>Name</td><td>Type</td><td>Created Date</td><td>Created By</td><td>Modified Date</td><td>Modified By</td><td>Size</td><td>Change Count</td><td>ChangeToken</td><td>ACM</td></tr>`)
	objects := response.Objects
	for idx := range objects {
		object := objects[idx]
		fmt.Fprintf(w, "<tr>")
		switch {
		case object.TypeName.String == "Folder":
			fmt.Fprintf(
				w,
				"<td><a href='%s/object/%s/list'>%s</a></td>",
				config.RootURL,
				hex.EncodeToString(object.ID),
				object.Name,
			)
		default:
			fmt.Fprintf(
				w,
				"<td><a href='%s/object/%s/stream'>%s</a></td>",
				config.RootURL,
				hex.EncodeToString(object.ID),
				object.Name,
			)
		}
		fmt.Fprintf(w, "<td>%s</td>", object.TypeName.String)
		fmt.Fprintf(w, "<td>%s</td>", GetFormattedDate(object.CreatedDate))
		fmt.Fprintf(w, "<td>%s</td>", config.GetCommonName(object.CreatedBy))
		fmt.Fprintf(w, "<td>%s</td>", GetFormattedDate(object.ModifiedDate))
		fmt.Fprintf(w, "<td>%s</td>", config.GetCommonName(object.ModifiedBy))
		fmt.Fprintf(w, "<td>%d</td>", object.ContentSize.Int64)
		fmt.Fprintf(w, "<td>%d</td>", object.ChangeCount)
		fmt.Fprintf(w, "<td>%s</td>", object.ChangeToken)
		fmt.Fprintf(w, "<td>%s</td>", object.RawAcm.String)
		fmt.Fprintf(w, "</tr>")
	}
	fmt.Fprintf(w, "</table>")

	// Finally, provide a form to add a new child folder here
	if canCreateFolder {
		fmt.Fprintf(w, createFileForm, config.RootURL, hex.EncodeToString(parentObject.ID))
		fmt.Fprintf(w, createObjectForm, config.RootURL, hex.EncodeToString(parentObject.ID))
	}
}

func createPagerAsHTML(baseURI string, PageCount int, PageNumber int, PageSize int) string {
	var o string
	o += "<table id='listObjectsPager'><tr>"
	for pc := 1; pc <= PageCount; pc++ {
		if pc == PageNumber {
			o += "<td width='50' align='center' bgcolor='grey'>&nbsp;"
			o += strconv.Itoa(pc)
		} else {
			o += "<td width='50' align='center'>&nbsp;"
			o += "<a href='" + baseURI + "?PageNumber=" + strconv.Itoa(pc) + "&PageSize=" + strconv.Itoa(PageSize) + "'>" + strconv.Itoa(pc) + "</a>"
		}
		o += "&nbsp;</td>"
	}
	o += "</tr></table>"
	return o
}

//Temporary goo before JSON libs
var createFileForm = `
<hr/>
<h2>Create Folder</h2>
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
`

//Temporary goo before JSON libs
var createObjectForm = `
<hr/>
<h2>Create File</h2>
<form method="post" action="%s/object" enctype="multipart/form-data">
<table>
	<input type="hidden" name="parentId" value="%s"/>
	<input type="hidden" name="async" value="true"/>
	<tr>
		<td>Object Name</td>
		<td><input type="text" id="title" name="title" /></td>
	</tr>
	<tr>
		<td>Type</td>
		<td><select id="type" name="type">
				<option value="File">File</option>
				<option value="Folder">Folder</option>
				</select>
		</td>
	</tr>
	<tr>
		<td>Classification</td>
		<td><select id="classification" name="classification">
				<option value='U'>UNCLASSIFIED</option>
				<option value='C'>CLASSIFIED</option>
				<option value='S'>SECRET</option>
				<option value='T'>TOP SECRET</option>
				</select>
		</td>
	</tr>
	<tr>
		<td>File Content</td>
		<td><input type="file" name="filestream" /></td>
	</tr>
</table>
<input type="submit" value="Upload" />
</form>
	`

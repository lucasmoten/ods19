package server

import (
	//"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/metadata/models"
)

var createObjectForm = `
<hr/>
<form method="post" action="%s/object" enctype="multipart/form-data">
<table>
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

<hr />
Values received
<br />
title: %s
<br />
type: %s
<br />
classification: %s
	`

// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
) {

	var obj models.ODObject
	var acm models.ODACM
	var grant models.ODObjectPermission
	var err error

	if r.Method == "POST" {

		//	grant.ObjectID = obj.ID
		grant.Grantee = caller.DistinguishedName
		grant.AllowRead = true
		grant.AllowCreate = true
		grant.AllowUpdate = true
		grant.AllowDelete = true

		// Set creator
		obj.CreatedBy = caller.DistinguishedName
		acm.CreatedBy = caller.DistinguishedName

		rName := createRandomName()
		fileKey, iv := createKeyIVPair()
		obj.ContentConnector.String = rName
		obj.EncryptIV = iv
		grant.EncryptKey = fileKey
		h.acceptObjectUpload(w, r, caller, &obj, &acm, &grant)

		obj.Permissions = make([]models.ODObjectPermission, 1)
		obj.Permissions[0] = grant

		err = h.DAO.CreateObject(&obj, &acm)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "error storing object")
			return
		}
		log.Printf("server created object:%v with contentConnector:%v", obj.ODID, obj.ContentConnector.String)
	}

	if r.Method == "POST" {
		//TODO: json response rendering
		w.Header().Set("Content-Type", "application/json")
		link := GetObjectLinkFromObject(config.RootURL, &obj)
		//Write a link back to the user so that it's possible to do an update on this object
		encoder := json.NewEncoder(w)
		encoder.Encode(&link)
	} else {
		//Push all of the html stuff in one place, so that we can eliminate it
		//when we have a real user interface that uses the API
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, pageTemplateStart, "createObject", caller.DistinguishedName)
		fmt.Fprintf(w, createObjectForm, config.RootURL, obj.Name, obj.TypeName.String, acm.Classification.String)
		fmt.Fprintf(w, pageTemplateEnd)
	}
}

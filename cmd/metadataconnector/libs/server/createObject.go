package server

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "createObject", caller.DistinguishedName)

	fmt.Fprintf(w, `
	<hr />
	<form method="post" action="/services/object-drive/object" enctype="multipart/form-data">
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
					<option>UNCLASSIFIED</option>
					</select>
			</td>
		<tr>
			<td>File Content</td>
			<td><input type="file" name="filestream" /></td>
		</tr>
	</table>
	<input type="submit" value="Upload" />
	</form>

			`)

	if r.Method == "POST" {
		var obj models.ODObject
		var acm models.ODACM

		// Set creator
		obj.CreatedBy = caller.DistinguishedName
		acm.CreatedBy = caller.DistinguishedName

		// TODO: Access Check

		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			panic(err)
		} // if err != nil
		for {
			part, err := multipartReader.NextPart()
			if err != nil {
				if err == io.EOF {
					break //just an eof...not an error
				} else {
					h.sendErrorResponse(w, 500, err, "error getting a part")
					return
				}
			} // if err != nil

			switch {
			case part.FormName() == "title":
				obj.Name = getFormValueAsString(part)
			case part.FormName() == "type":
				obj.TypeName.String = getFormValueAsString(part)
				obj.TypeName.Valid = (len(obj.TypeName.String) > 0)
			case part.FormName() == "classification":
				acm.Classification.String = getFormValueAsString(part)
				acm.Classification.Valid = (len(acm.Classification.String) > 0)
			case len(part.FileName()) > 0:
				if len(obj.Name) == 0 {
					obj.Name = part.FileName()
					// TODO: Drain file to temporary local space and then puh to S3
					// TODO: Capture info needed into obj
					//			obj.ContentConnector (should define S3 + bucketName)
					//			obj.EncyrptIV
					//			obj.EncyrptKey
					//			obj.ContentSize
				} // if len(obj.Name) == 0 {
			} // switch
		} //for

		// TODO: Validation

		// TODO: add object to database
		dao.CreateObject(h.MetadataDB, &obj, &acm)

		fmt.Fprintf(w, `
		<hr />
Values received
<br />
title: %s
<br />
type: %s
<br />
classification: %s
			`, obj.Name, obj.TypeName.String, acm.Classification.String)
	} // if r.Method == "POST" {

	fmt.Fprintf(w, pageTemplateEnd)

}

// getFormValueAsString reads a multipart value into a limited length byte
// array and returns it.
// TODO: Move to a utility file since this is useful for all other requests
// doing multipart.
// TODO: This effectively limits the acceptable length of a field to 1KB which
// is too restrictive for certain values (lengthy descriptions, abstracts, etc)
// which will need revisited
func getFormValueAsString(part *multipart.Part) string {
	valueAsBytes := make([]byte, 1024)
	n, err := part.Read(valueAsBytes)
	if err != nil {
		if err == io.EOF {
			return ""
		} else {
			panic(err)
		}
	} // if err != nil
	return string(valueAsBytes[0:n])
}

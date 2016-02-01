package server

import (
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
)

func (h AppServer) drainFileToS3(
	bucket *string,
	rName string,
) error {
	sess := h.AWSSession
	outFileUploaded := h.CacheLocation + "/" + rName + ".uploaded"

	fIn, err := os.Open(outFileUploaded)
	if err != nil {
		log.Printf("Cant drain off file: %v", err)
		return err
	}
	defer fIn.Close()
	log.Printf("draining to S3 %s: %s", *bucket, rName)

	uploader := s3manager.NewUploader(sess)
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   fIn,
		Bucket: bucket,
		Key:    aws.String(h.CacheLocation + "/" + rName),
	})
	if err != nil {
		log.Printf("Could not write to S3: %v", err)
		return err
	}

	//Rename the file to note that it only lives here as cached for download
	//It might be deleted at any time
	outFileCached := h.CacheLocation + "/" + rName + ".cached"
	err = os.Rename(outFileUploaded, outFileCached)
	if err != nil {
		log.Printf("Unable to rename uploaded file %s %v:", outFileUploaded, err)
		return err
	}

	log.Printf("Uploaded to %v: %v", *bucket, result.Location)
	return err
}

func (h AppServer) beginUpload(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	part *multipart.Part,
	obj models.ODObject,
	acm models.ODACM,
) (grant models.ODObjectPermission, err error) {
	if _, err = os.Stat(h.CacheLocation); os.IsNotExist(err) {
		err = os.Mkdir(h.CacheLocation, 0700)
		log.Printf("Creating cache directory %s", h.CacheLocation)
		if err != nil {
			log.Printf("Cannot create cache directory: %v", err)
			return grant, err
		}
	}
	//Make up a random name for our file - don't deal with versioning yet
	rName := createRandomName()
	outFileUploading := h.CacheLocation + "/" + rName + ".uploading"
	outFileUploaded := h.CacheLocation + "/" + rName + ".uploaded"

	outFile, err := os.Create(outFileUploading)
	if err != nil {
		log.Printf("Unable to open ciphertext uploading file %s %v:", outFileUploading, err)
		return grant, err
	}
	defer outFile.Close()

	//Generate random key and iv
	key, iv := createKeyIVPair()

	//Write the encrypted data to the filesystem
	checksum, length, err := doCipherByReaderWriter(part, outFile, key, iv)
	if err != nil {
		log.Printf("Unable to write ciphertext %s %v:", outFileUploading, err)
		return grant, err
	}
	//Rename it to indicate that it can be moved to S3
	err = os.Rename(outFileUploading, outFileUploaded)
	if err != nil {
		log.Printf("Unable to rename uploaded file %s %v:", outFileUploading, err)
		return grant, err
	}

	//Record metadata
	obj.ContentConnector.String = rName
	obj.ContentHash.String = hex.EncodeToString(checksum)
	obj.ContentSize.Int64 = length
	obj.EncryptIV = iv
	log.Printf("TODO: trying to create a grant when I don't yet know the objectID")
	//	grant.ObjectID = obj.ID
	grant.Grantee = caller.DistinguishedName
	grant.AllowRead = true
	grant.AllowCreate = true
	grant.AllowUpdate = true
	grant.AllowDelete = true
	grant.EncryptKey = key
	//Uploaded file is effectively enqueued for S3 upload.
	//Go ugly early, and just make this drain-off a goroutine
	//We may need to have a limited number of concurrent uploads,
	//with something like Fair-Weighted-Queueing to pick the order
	//in which they happen (ie: look at file sizes and create timestamps)
	go h.drainFileToS3(aws.String("decipherers"), rName)
	return grant, err
}

// createObject is a method handler on AppServer for createObject microservice
// operation.
func (h AppServer) createObject(w http.ResponseWriter, r *http.Request, caller Caller) {

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "createObject", caller.DistinguishedName)

	fmt.Fprintf(w, `
	<hr />
	<form method="post" action="https://twl-server-generic2:8080/service/metadataconnector/1.0/object" enctype="multipart/form-data">
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
		var grant models.ODObjectPermission
		var err error

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
				grant, err = h.beginUpload(w, r, caller, part, obj, acm)
				if err != nil {
					h.sendErrorResponse(w, 500, err, "error caching file")
					return
				}
			} // switch
		} //for

		// TODO: Validation

		// TODO: add object to database
		obj.Permissions = make([]models.ODObjectPermission, 1)
		obj.Permissions[0] = grant

		err = dao.CreateObject(h.MetadataDB, &obj, &acm)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "error storing object")
			return
		}
		//log.Printf("TODO: add grant permission: %v", grant)

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

package server

import (
	//"encoding/hex"
	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/performance"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
)

func (h AppServer) drainFileToS3(
	bucket *string,
	rName string,
) error {
	beganAt := h.Tracker.BeginTime(performance.S3DrainTo)
	err := h.drainFileToS3Timed(bucket, rName)
	stat, cachedErr := os.Stat(h.CacheLocation + "/" + rName + ".cached")
	if cachedErr != nil {
		log.Printf("could not get length of cached file %s", rName)
	}
	length := stat.Size()

	h.Tracker.EndTime(
		performance.S3DrainTo,
		beganAt,
		performance.SizeJob(length),
	)
	return err
}

func (h AppServer) drainFileToS3Timed(
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

func extIs(name string, ext string) bool {
	return strings.ToLower(path.Ext(name)) == strings.ToLower(ext)
}

//I'm sure there is a function call somewhere with this database....
func guessContentType(name string) string {
	contentType := "text/plain"
	switch {
	case extIs(name, ".htm"):
		contentType = "text/html"
	case extIs(name, ".html"):
		contentType = "text/html"
	case extIs(name, ".txt"):
		contentType = "text"
	case extIs(name, ".mp3"):
		contentType = "audio/mp3"
	case extIs(name, ".jpg"):
		contentType = "image/jpeg"
	case extIs(name, ".png"):
		contentType = "image/png"
	case extIs(name, ".gif"):
		contentType = "image/gif"
	case extIs(name, ".m4v"):
		contentType = "video/mp4"
	case extIs(name, ".mp4"):
		contentType = "video/mp4"
	case extIs(name, ".mov"):
		contentType = "video/mov"
	default:
		ext := path.Ext(name)
		if len(ext) > 1 {
			contentType = fmt.Sprintf("application/%s", ext[1:])
		}
	}
	log.Printf("uploaded %s as %s", name, contentType)
	return contentType
}

func (h AppServer) beginUpload(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	acm *models.ODACM,
	grant *models.ODObjectPermission,
) (err error) {

	beganAt := h.Tracker.BeginTime(performance.UploadCounter)
	err = h.beginUploadTimed(w, r, caller, part, obj, acm, grant)

	h.Tracker.EndTime(
		performance.UploadCounter,
		beganAt,
		performance.SizeJob(obj.ContentSize.Int64),
	)

	return err
}

func (h AppServer) beginUploadTimed(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	acm *models.ODACM,
	grant *models.ODObjectPermission,
) (err error) {
	rName := obj.ContentConnector.String
	iv := obj.EncryptIV
	fileKey := grant.EncryptKey

	err = h.CacheMustExist()
	if err != nil {
		return err
	}
	//Make up a random name for our file - don't deal with versioning yet
	outFileUploading := h.CacheLocation + "/" + rName + ".uploading"
	outFileUploaded := h.CacheLocation + "/" + rName + ".uploaded"

	outFile, err := os.Create(outFileUploading)
	if err != nil {
		log.Printf("Unable to open ciphertext uploading file %s %v:", outFileUploading, err)
		return err
	}
	defer outFile.Close()

	//Write the encrypted data to the filesystem
	checksum, _, err := doCipherByReaderWriter(part, outFile, fileKey, iv)
	if err != nil {
		log.Printf("Unable to write ciphertext %s %v:", outFileUploading, err)
		return err
	}

	//Scramble the fileKey with the masterkey - will need it once more on retrieve
	applyPassphrase(h.MasterKey+caller.DistinguishedName, fileKey)

	//Rename it to indicate that it can be moved to S3
	err = os.Rename(outFileUploading, outFileUploaded)
	if err != nil {
		log.Printf("Unable to rename uploaded file %s %v:", outFileUploading, err)
		return err
	}

	stat, err := os.Stat(outFileUploaded)
	if err != nil {
		log.Printf("Unable to get stat on uploaded file %s: %v", outFileUploaded, err)
	}

	//Record metadata
	obj.ContentHash = checksum
	obj.ContentSize.Int64 = stat.Size()
	//XXX making the S3 upload synchronous temporarily, in the absence of a
	//mechanism for knowing whether the file has made it up to S3 yet
	h.drainFileToS3(aws.String("decipherers"), rName)
	return err
}

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

func (h AppServer) acceptObjectUpload(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	obj *models.ODObject,
	acm *models.ODACM,
	grant *models.ODObjectPermission,
) {
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
			//XXX We just have a small set of objects that map to raw acm at the moment
			obj.RawAcm.String = h.Classifications[acm.Classification.String]
		case len(part.FileName()) > 0:
			//Guess the content type and name
			obj.ContentType.String = guessContentType(part.FileName())
			if obj.Name == "" {
				obj.Name = part.FileName()
			}
			err = h.beginUpload(w, r, caller, part, obj, acm, grant)
			if err != nil {
				h.sendErrorResponse(w, 500, err, "error caching file")
				return
			}
		} // switch
	} //for
}

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

		err = dao.CreateObject(h.MetadataDB, &obj, &acm)
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
		}
		panic(err)
	} // if err != nil
	return string(valueAsBytes[0:n])
}

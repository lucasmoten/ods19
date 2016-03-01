package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/performance"
	"decipher.com/oduploader/protocol"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func (h AppServer) acceptObjectUpload(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	obj *models.ODObject,
	acm *models.ODACM,
	grant *models.ODObjectPermission,
) {
	multipartReader, err := r.MultipartReader()
	if err != nil {
		panic(err)
	}
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
		case part.FormName() == "CreateObjectRequest":
			s := getFormValueAsString(part)
			//It's the same as the database object, but this function might be
			//dealing with a retrieved object, so we get fields individually
			var createObjectRequest protocol.Object
			err := json.Unmarshal([]byte(s), &createObjectRequest)
			if err != nil {
				h.sendErrorResponse(w, 400, err, "Could not decode CreateObjectRequest.")
			}
			err = mapping.OverwriteODObjectWithProtocolObject(obj, &createObjectRequest)
			if err != nil {
				h.sendErrorResponse(w, 400, err, "Could not extract data from json response")
				return
			}
		case len(part.FileName()) > 0:
			//Guess the content type and name if it wasn't supplied
			if obj.ContentType.Valid == false || len(obj.ContentType.String) == 0 {
				obj.ContentType.String = guessContentType(part.FileName())
			}
			if obj.Name == "" {
				obj.Name = part.FileName()
			}
			async := true
			err = h.beginUpload(w, r, caller, part, obj, acm, grant, async)
			if err != nil {
				h.sendErrorResponse(w, 500, err, "error caching file")
				return
			}
		} // switch
	} //for
}

func (h AppServer) beginUpload(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	acm *models.ODACM,
	grant *models.ODObjectPermission,
	async bool,
) (err error) {

	beganAt := h.Tracker.BeginTime(performance.UploadCounter)
	err = h.beginUploadTimed( /*w, r,*/ caller, part, obj, acm, grant, async)

	h.Tracker.EndTime(
		performance.UploadCounter,
		beganAt,
		performance.SizeJob(obj.ContentSize.Int64),
	)

	return err
}

func (h AppServer) beginUploadTimed(
	//	w http.ResponseWriter,
	//	r *http.Request,
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	acm *models.ODACM,
	grant *models.ODObjectPermission,
	async bool,
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
	checksum, length, err := doCipherByReaderWriter(part, outFile, fileKey, iv, "uploading from browser")
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
	log.Printf("rename:%s -> %s", outFileUploading, outFileUploaded)

	//Record metadata
	obj.ContentHash = checksum
	obj.ContentSize.Int64 = length

	//Just return 200 when we run async, because the client tells
	//us whether it's async or not.
	if async {
		go h.drainFileToS3(aws.String(config.DefaultBucket), rName, length, 3)
	} else {
		h.drainFileToS3(aws.String(config.DefaultBucket), rName, length, 3)
	}
	return err
}

//We get penalized on throughput if these fail a lot.
//I think that's reasonable to be measuring "goodput" this way.
func (h AppServer) drainFileToS3(
	bucket *string,
	rName string,
	size int64,
	tries int,
) error {
	beganAt := h.Tracker.BeginTime(performance.S3DrainTo)
	err := h.drainFileToS3Attempt(bucket, rName, size, tries)
	h.Tracker.EndTime(
		performance.S3DrainTo,
		beganAt,
		performance.SizeJob(size),
	)
	return err
}

func (h AppServer) drainFileToS3Attempt(
	bucket *string,
	rName string,
	size int64,
	tries int,
) error {
	err := h.drainFileToS3Timed(bucket, rName, size)
	tries = tries - 1
	if err != nil {
		//The problem is that we get things like transient DNS errors,
		//after we took custody of the file.  We will need something
		//more robust than this eventually.  We have the file, while
		//not being uploaded if all attempts fail.
		log.Printf("unable to drain file to S3.  Trying again:%v", err)
		if tries > 0 {
			err = h.drainFileToS3Attempt(bucket, rName, size, tries)
		}
	}
	return err
}

func (h AppServer) drainFileToS3Timed(
	bucket *string,
	rName string,
	size int64,
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
	log.Printf("rename:%s -> %s", outFileUploaded, outFileCached)

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

func (h AppServer) transferFileFromS3(
	bucket *string,
	theFile string,
	length int64,
) {
	beganAt := h.Tracker.BeginTime(performance.S3DrainFrom)
	h.transferFileFromS3Timed(bucket, theFile)

	h.Tracker.EndTime(
		performance.S3DrainFrom,
		beganAt,
		performance.SizeJob(length),
	)
}

func (h AppServer) transferFileFromS3Timed(
	bucket *string,
	theFile string,
) {
	log.Printf("Get from S3 bucket %s: %s", *bucket, theFile)
	foutCaching := h.CacheLocation + "/" + theFile + ".caching"
	foutCached := h.CacheLocation + "/" + theFile + ".cached"
	fOut, err := os.Create(foutCaching)
	if err != nil {
		log.Printf("Unable to write local buffer file %s: %v", theFile, err)
	}
	defer fOut.Close()

	downloader := s3manager.NewDownloader(h.AWSSession)
	_, err = downloader.Download(
		fOut,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    aws.String(h.CacheLocation + "/" + theFile),
		},
	)
	if err != nil {
		log.Printf("Unable to download out of S3 bucket %v: %v", *bucket, theFile)
	}
	//Signal that we finally cached the file
	err = os.Rename(foutCaching, foutCached)
	if err != nil {
		log.Printf("Failed to rename from %s to %s", foutCaching, foutCached)
	}
	log.Printf("rename:%s -> %s", foutCaching, foutCached)
}

package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"encoding/hex"
	//"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	//"path"
	"regexp"
	"strconv"
	//"strings"
)

func (h AppServer) dumpCacheLocation() {
	//XXX temp hack just so we can see in the container.
	//Files is an array, which is bad
	//if the queue is large, or accessed often!!!
	files, err := ioutil.ReadDir(h.CacheLocation)
	if err != nil {
		log.Printf("Error reading cache dir:%v", err)
		return
	}
	for _, f := range files {
		log.Printf("%s", f.Name())
	}
}

func (h AppServer) transferFileFromS3(
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
}

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
		////The AAC check function exists.  I'm not sure how to call it here yet
		////without the acm and token type already given.
		//if h.AAC.CheckAccess(caller.DistinguishedName, "pki-dias", acm) {
		// canRetrieve = true
		//}
		canRetrieve = true
	}
	// TODO Check object permission grants
	/////note... can't decrypt the stream without the grant.

	if !canRetrieve {
		h.sendErrorResponse(w, 403, nil, "Caller does not have permission to the requested object")
	}

	// TODO: Based upon object metadata, get the object from S3
	//		object.ContentConnector
	//		object.ContentHash
	if _, err = os.Stat(h.CacheLocation); os.IsNotExist(err) {
		err = os.Mkdir(h.CacheLocation, 0700)
		log.Printf("Creating cache directory %s", h.CacheLocation)
		if err != nil {
			log.Printf("Cannot create cache directory: %v", err)
			return
		}
	}
	h.dumpCacheLocation()

	//hasStream := false
	cipherTextS3Name := h.CacheLocation + "/" + object.ContentConnector.String
	cipherTextName := cipherTextS3Name + ".cached"
	_, err = os.Stat(cipherTextName)
	if err == nil {
		log.Printf("file exists and is cached as: %s", object.ContentConnector.String)
		//hasStream = true
	} else {
		if os.IsNotExist(err) {
			log.Printf("file is not cached.  Caching it now.")
			bucket := aws.String("decipherers")
			//When this finishes, cipherTextName should exist.  It could take a
			//very long time though.
			h.transferFileFromS3(bucket, cipherTextS3Name)
		}
	}
	var key []byte
	iv := object.EncryptIV
	if len(object.Permissions) == 0 {
		log.Printf("We can't decrypt files that don't have permissions!")
		//hasStream = false
	} else {
		key = object.Permissions[0].EncryptKey
	}

	cipherText, err := os.Open(cipherTextName)
	if err != nil {
		log.Printf("Unable to open ciphertext %s:%v", cipherTextName, err)
		return
	}
	defer cipherText.Close()

	//Sanity check to dump metadata about what is currently being downloaded
	log.Printf(
		"decrypt with iv:%v key:%v type:%v sz:%v",
		iv,
		key,
		object.ContentType.String,
		object.ContentSize.Int64,
	)

	w.Header().Set("Content-Type", object.ContentType.String)
	if object.ContentSize.Valid && object.ContentSize.Int64 > int64(0) {
		w.Header().Set("Content-Length", strconv.FormatInt(object.ContentSize.Int64, 10))
	}

	//Actually send back the ciphertext
	_, _, err = doCipherByReaderWriter(
		cipherText,
		w,
		key,
		iv,
	)

	if err != nil {
		log.Printf("error sending decrypted ciphertext %s:%v", cipherTextName, err)
		return
	}
	/*
		//XXX should be returning an http error code in this case
		if !hasStream {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, pageTemplateStart, "getObjectStream", caller.DistinguishedName)
			fmt.Fprintf(w, "No content")
			fmt.Fprintf(w, pageTemplateEnd)
			return
		}
	*/

	/*
		fmt.Fprintf(w, pageTemplateStart, "getObjectStream", caller.DistinguishedName)
		fmt.Fprintf(w, pageTemplateEnd)
	*/
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

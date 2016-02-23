package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"encoding/hex"
	"encoding/json"
	//"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	//"path"
	"regexp"
	"strconv"
	//"strings"
	"decipher.com/oduploader/performance"
)

func (h AppServer) getObjectStreamObject(w http.ResponseWriter, r *http.Request, caller Caller) (*models.ODObject, error) {
	// Identify requested object
	objectID := getIDOfObjectTORetrieveStream(r.URL.RequestURI())
	// If not valid, return
	if objectID == "" {
		h.sendErrorResponse(w, 400, nil, "URI provided by caller does not specify an object identifier")
		return nil, nil
	}
	// Convert to byte
	objectIDByte, err := hex.DecodeString(objectID)
	if err != nil {
		h.sendErrorResponse(w, 400, nil, "Identifier provided by caller is not a hexidecimal string")
		return nil, err
	}
	// Retrieve from database
	var objectRequested models.ODObject
	objectRequested.ID = objectIDByte
	object, err := h.DAO.GetObject(&objectRequested, false)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cannot get object")
		return nil, err
	}
	return object, nil
}

/*
  We are wrapping around getting object streams to time them.
	TODO: This is including cache miss time.
*/
func (h AppServer) getObjectStream(w http.ResponseWriter, r *http.Request, caller Caller) {
	object, err := h.getObjectStreamObject(w, r, caller)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cannot get object")
		return
	}
	if object == nil {
		log.Printf("did not find an object")
		h.sendErrorResponse(w, 500, err, "did not get object")
		return
	}

	beganAt := h.Tracker.BeginTime(performance.DownloadCounter)

	h.getObjectStreamWithObject(w, r, caller, object)

	h.Tracker.EndTime(
		performance.DownloadCounter,
		beganAt,
		performance.SizeJob(object.ContentSize.Int64),
	)
}

func (h AppServer) getObjectStreamWithObject(w http.ResponseWriter, r *http.Request, caller Caller, object *models.ODObject) {
	var err error
	var err2 error
	var err3 error

	// Get the key from the permission
	var permission *models.ODObjectPermission
	var fileKey []byte
	iv := object.EncryptIV

	if len(object.Permissions) == 0 {
		log.Printf("We can't decrypt files that don't have permissions!")
		h.sendErrorResponse(w, 403, nil, "No permission for file")
		return
	}

	//XXX watch for very large number of permissions on a file!
	for _, v := range object.Permissions {
		permission = &v
		if permission.AllowRead && permission.Grantee == caller.DistinguishedName {
			fileKey = permission.EncryptKey
			//Unscramble the fileKey with the masterkey - will need it once more on retrieve
			applyPassphrase(h.MasterKey+caller.DistinguishedName, fileKey)
		}
	}

	////Clean out the permission if aac check fails
	tokenType := "pki_dias"
	dn := caller.DistinguishedName
	aacResponse, err := h.AAC.CheckAccess(dn, tokenType, object.RawAcm.String)
	if err != nil {
		log.Printf(
			"AAC not responding to checkaccess for %s:%s:%s:%v",
			dn, tokenType, object.RawAcm.String, err,
		)
		permission = nil
	}
	if aacResponse == nil {
		log.Printf(
			"AAC null response for checkaccess for %s:%s:%s:%v",
			dn, tokenType, object.RawAcm.String, err,
		)
		permission = nil
	} else {
		log.Printf("AAC grants access to %s for %s", dn, object.RawAcm.String)
	}

	if permission == nil {
		h.sendErrorResponse(w, 403, nil, "No permission for file")
		return
	}

	// Database checks and AAC checks take time, particularly AAC.
	// We may have just uploaded this file, with it still draining
	// to S3 in the background.  So open the file as late as possible
	// so that we take advantage of that parallelism.

	//Make sure that the cache exists
	err = h.CacheMustExist()
	if err != nil {
		return
	}

	//Fall back on the uploaded copy for download if we need to
	var cipherText *os.File
	cipherTextS3Name := h.CacheLocation + "/" + object.ContentConnector.String
	cipherTextName := cipherTextS3Name + ".cached"
	cipherTextUploadedName := cipherTextS3Name + ".uploaded"

	//Try to find the cached file
	if cipherText, err = os.Open(cipherTextName); err != nil {
		if os.IsNotExist(err) {
			//Try the file being uploaded into S3
			if cipherText, err2 = os.Open(cipherTextUploadedName); err2 != nil {
				if os.IsNotExist(err2) {
					//Maybe it's cached now?
					if cipherText, err3 = os.Open(cipherTextName); err3 != nil {
						if os.IsNotExist(err3) {
							//File is really not cached or uploaded.
							//If it's caching, it's partial anyway
							//leave cipherText nil, and wait for re-cache
						} else {
							//Some other error.  Pretend it's permissions
							h.sendErrorResponse(w, 403, err, "No permission for file")
							return
						}
					} else {
						//cached file exists
					}
				} else {
					//Some other error.  Pretend it's permissions
					h.sendErrorResponse(w, 403, err, "No permission for file")
					return
				}
			} else {
				//uploaded file exists.  use it.
				log.Printf("using uploaded file")
			}
		} else {
			//Some other error.  Pretend it's permissions
			h.sendErrorResponse(w, 403, err, "No permission for file")
			return
		}
	} else {
		//the cached file exists
	}

	//We have no choice but to recache and wait
	if cipherText == nil {
		log.Printf("file is not cached.  Caching it now.")
		bucket := aws.String("decipherers")
		//When this finishes, cipherTextName should exist.  It could take a
		//very long time though.
		//XXX blocks for a long time - maybe we should return an error code and
		// an estimate of WHEN to retry in this case
		h.transferFileFromS3(bucket, object.ContentConnector.String)
		if cipherText, err = os.Open(cipherTextName); err != nil {
			h.sendErrorResponse(w, 403, err, "No permission for file")
			return
		}
	}

	//We have the file and it's open.  Be sure to close it.
	defer cipherText.Close()

	w.Header().Set("Content-Type", object.ContentType.String)
	if object.ContentSize.Valid && object.ContentSize.Int64 > int64(0) {
		w.Header().Set("Content-Length", strconv.FormatInt(object.ContentSize.Int64, 10))
	}

	//A visibility hack, so that I can see metadata about the object from a GET
	//This lets you look in a browser and check attributes on an object that came
	//back.
	objectLink := mapping.MapODObjectToObject(object)
	objectLinkAsJSONBytes, err := json.Marshal(objectLink)
	if err != nil {
		log.Printf("Unable to marshal object metadata:%v", err)
	}
	objectLinkAsJSON := string(objectLinkAsJSONBytes)
	w.Header().Set("Object-Data", objectLinkAsJSON)

	//Actually send back the ciphertext
	_, _, err = doCipherByReaderWriter(
		cipherText,
		w,
		fileKey,
		iv,
		"client downloading",
	)

	if err != nil {
		log.Printf("error sending decrypted ciphertext %s:%v", cipherTextName, err)
		return
	}

	//Update the timestamps to note the last time it was used
	tm := time.Now()
	os.Chtimes(cipherTextName, tm, tm)
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

package server

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strconv"
	"time"
    "fmt"
	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/performance"
	"github.com/aws/aws-sdk-go/aws"
)

func (h AppServer) getObjectStreamObject(w http.ResponseWriter, r *http.Request, caller Caller) (models.ODObject, *AppError, error) {
	var object models.ODObject
	// Identify requested object
	objectID := getIDOfObjectTORetrieveStream(r.URL.Path)
	// If not valid, return
	if objectID == "" {
        msg := "URI provided by caller does not specify an object identifier"
        return object, &AppError{400, nil, msg},nil
	}
	// Convert to byte
	objectIDByte, err := hex.DecodeString(objectID)
	if err != nil {
        msg := "Identifier provided by caller is not a hexidecimal string"
		return object, &AppError{400, err, msg},err
	}
	// Retrieve from database
	var objectRequested models.ODObject
	objectRequested.ID = objectIDByte
	object, err = h.DAO.GetObject(objectRequested, false)
	if err != nil {
        msg := "cannot get object"
		return object, &AppError{500, err, msg}, err
	}
	return object, nil, nil
}

/*
  We are wrapping around getting object streams to time them.
	TODO: This is including cache miss time.
*/
func (h AppServer) getObjectStream(w http.ResponseWriter, r *http.Request, caller Caller) {
	req, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("unable to dump http request:%v", err)
	} else {
		log.Printf("%s", string(req))
	}

	object, herr, err := h.getObjectStreamObject(w, r, caller)
    if herr != nil {
        h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
        return
    }
	if err != nil {
		h.sendErrorResponse(w, 500, err, "cannot get object")
		return
	}
	if object.ID == nil {
		log.Printf("did not find an object")
		h.sendErrorResponse(w, 500, err, "did not get object")
		return
	}

	beganAt := h.Tracker.BeginTime(performance.DownloadCounter)
	defer h.Tracker.EndTime(
		performance.DownloadCounter,
		beganAt,
		performance.SizeJob(object.ContentSize.Int64),
	)

	if herr := h.getObjectStreamWithObject(w, r, caller, object); herr != nil {
        h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
    }

}

func (h AppServer) getObjectStreamWithObject(w http.ResponseWriter, r *http.Request, caller Caller, object models.ODObject) (*AppError) {
	var err error
	var err2 error
	var err3 error

	// Get the key from the permission
	var permission *models.ODObjectPermission
	var fileKey []byte
	iv := object.EncryptIV

	if len(object.Permissions) == 0 {
        return &AppError{403, fmt.Errorf("We cannot decrypt files lacking permissions"),"Unauthorized"}
	}

	if object.IsDeleted {
		switch {
		case object.IsExpunged:
			return &AppError{410, err, "The object no longer exists."}
		case object.IsAncestorDeleted:
			return &AppError{405, err, "The object cannot be modified because an ancestor is deleted."}
		default:
			return &AppError{405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it."}
		}
	}    
	//XXX watch for very large number of permissions on a file!
	for _, v := range object.Permissions {
		permission = &v
		if permission.AllowRead && permission.Grantee == caller.DistinguishedName {
			fileKey = permission.EncryptKey
			//Unscramble the fileKey with the masterkey - will need it once more on retrieve
			utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, fileKey)
			// Once we have a match, quit looking and avoid reapplying passphrase
			break
		}
	}

	////Clean out the permission if aac check fails
	if true {
		tokenType := "pki_dias"
		dn := caller.DistinguishedName
		log.Printf("Waiting for AAC to respond to %s for %s", dn, object.RawAcm.String)
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
	}

	if permission == nil {
		return &AppError{403, nil, "No permission for file"}
	}

	// Database checks and AAC checks take time, particularly AAC.
	// We may have just uploaded this file, with it still draining
	// to S3 in the background.  So open the file as late as possible
	// so that we take advantage of that parallelism.

	//Make sure that the cache exists
	err = h.CacheMustExist()
	if err != nil {
		return &AppError{500, err, "Our cache needs to exist"}
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
							return &AppError{403, err, "No permission for file"}
						}
					} else {
						//cached file exists
					}
				} else {
					//Some other error.  Pretend it's permissions
					return &AppError{403, err, "No permission for file"}
				}
			} else {
				//uploaded file exists.  use it.
				log.Printf("using uploaded file")
			}
		} else {
			//Some other error.  Pretend it's permissions
			return &AppError{403, err, "No permission for file"}
		}
	} else {
		//the cached file exists
	}

	//We have no choice but to recache and wait
	if cipherText == nil {
		log.Printf("file is not cached.  Caching it now.")
		bucket := aws.String(config.DefaultBucket)
		//When this finishes, cipherTextName should exist.  It could take a
		//very long time though.
		//XXX blocks for a long time - maybe we should return an error code and
		// an estimate of WHEN to retry in this case
		h.transferFileFromS3(bucket, object.ContentConnector.String, object.ContentSize.Int64)
		if cipherText, err = os.Open(cipherTextName); err != nil {
			return &AppError{403, err, "No permission for file"}
		}
	}

	//We have the file and it's open.  Be sure to close it.
	defer cipherText.Close()

	w.Header().Set("Content-Type", object.ContentType.String)
	if object.ContentSize.Valid && object.ContentSize.Int64 > int64(0) {
		w.Header().Set("Content-Length", strconv.FormatInt(object.ContentSize.Int64, 10))
	}
	w.Header().Set("Accept-Ranges", "none")

	//A visibility hack, so that I can see metadata about the object from a GET
	//This lets you look in a browser and check attributes on an object that came
	//back.
	objectLink := mapping.MapODObjectToObject(&object)
	objectLinkAsJSONBytes, err := json.Marshal(objectLink)
	if err != nil {
		log.Printf("Unable to marshal object metadata:%v", err)
	}
	objectLinkAsJSON := string(objectLinkAsJSONBytes)
	w.Header().Set("Object-Data", objectLinkAsJSON)

	//Actually send back the ciphertext
	_, _, err = utils.DoCipherByReaderWriter(
		cipherText,
		w,
		fileKey,
		iv,
		"client downloading",
	)

	if err != nil {
        return &AppError{
            Code:500, 
            Err:err, 
            Msg:fmt.Sprintf("error sending decrypted ciphertext (%s)",cipherTextName),
        }
	}

	//Update the timestamps to note the last time it was used
	tm := time.Now()
	os.Chtimes(cipherTextName, tm, tm)
    
    return nil
}

// getIDOfObjectTORetrieveStream accepts a passed in URI and finds whether an
// object identifier was passed within it for which the content stream is sought
func getIDOfObjectTORetrieveStream(uri string) string {
	re, _ := regexp.Compile("/object/([0-9a-fA-F]*)/stream")
	matchIndexes := re.FindStringSubmatchIndex(uri)
	if len(matchIndexes) == 0 {
		return ""
	}
	value := uri[matchIndexes[2]:matchIndexes[3]]
	return value
}

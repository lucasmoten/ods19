package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"net/url"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"
	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/performance"
)

/*
  We are wrapping around getting object streams to time them.
	TODO: This is including cache miss time.
*/
func (h AppServer) getObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("unable to dump http request:%v", err)
	} else {
		log.Printf("%s", string(req))
	}

	object, herr, err := retrieveObject(h.DAO, h.Routes.ObjectStream, r.URL.Path, true)
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

	if herr := h.getObjectStreamWithObject(ctx, w, r, object); herr != nil {
		h.sendErrorResponse(w, herr.Code, herr.Err, herr.Msg)
	}

}

func (h AppServer) getObjectStreamWithObject(ctx context.Context, w http.ResponseWriter, r *http.Request, object models.ODObject) *AppError {

	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return &AppError{500, err, "Could not determine user"}
	}

	// Get the key from the permission
	var permission *models.ODObjectPermission
	var fileKey []byte

	if len(object.Permissions) == 0 {
		return &AppError{403, fmt.Errorf("We cannot decrypt files lacking permissions"), "Unauthorized"}
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

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &object)
	if err != nil {
		return &AppError{500, err, "Error communicating with authorization service"}
	}
	if !hasAACAccess {
		return &AppError{403, err, "Unauthorized"}
	}

	// Fail fast: Don't even look at cache or retrieve if the file size is 0
	if !object.ContentSize.Valid || object.ContentSize.Int64 <= int64(0) {
		return &AppError{204, nil, "No content"}
	}

	// Database checks and AAC checks take time, particularly AAC.
	// We may have just uploaded this file, with it still draining
	// to S3 in the background.  So open the file as late as possible
	// so that we take advantage of that parallelism.
	h.getAndStreamFile(ctx, &object, w, fileKey, true)

	return nil
}

// This func broken out from the getObjectStream. It still needs refactored to
// be more maintainable and make use of an interface for the content streams
func (h AppServer) getAndStreamFile(ctx context.Context, object *models.ODObject, w http.ResponseWriter, encryptKey []byte, withMetadata bool) *AppError {
	var err error

	//Fall back on the uploaded copy for download if we need to
	var cipherFile *os.File
	cipherFileNameBasePath := h.DrainProvider.CacheLocation() + "/" + object.ContentConnector.String
	cipherFilePathCached := cipherFileNameBasePath + ".cached"
	cipherFilePathUploaded := cipherFileNameBasePath + ".uploaded"

	//Try to find the cached file
	if cipherFile, err = os.Open(cipherFilePathCached); err != nil {
		if os.IsNotExist(err) {
			//Try the file being uploaded into S3
			if cipherFile, err = os.Open(cipherFilePathUploaded); err != nil {
				if os.IsNotExist(err) {
					//Maybe it's cached now?
					if cipherFile, err = os.Open(cipherFilePathCached); err != nil {
						if os.IsNotExist(err) {
							//File is really not cached or uploaded.
							//If it's caching, it's partial anyway
							//leave cipherFile nil, and wait for re-cache
						} else {
							//Some other error.
							return &AppError{500, err, "Error opening file as cached state"}
						}
					} else {
						//cached file exists
					}
				} else {
					//Some other error.
					return &AppError{500, err, "Error opening file as uploaded state"}
				}
			} else {
				//uploaded file exists.  use it.
				log.Printf("using uploaded file")
			}
		} else {
			//Some other error.
			return &AppError{500, err, "Error opening file as initial cached state"}
		}
	} else {
		//the cached file exists
	}

	// Check if cipherFile was assigned, denoting whether or not pulling from cache
	if cipherFile == nil {
		// We have no choice but to recache and wait
		log.Printf("file is not cached.  Caching it now.")
		bucket := &config.DefaultBucket
		//When this finishes, cipherFilePathCached should exist.  It could take a
		//very long time though.
		//XXX blocks for a long time - maybe we should return an error code and
		// an estimate of WHEN to retry in this case
		// RECOMMEND: Break files in S3 greater then X size into 2 parts. One that is
		// of reasonable size to retrieve to get an initial output going, the other
		// consisting of the remainder. Only necessary for rather large files
		h.drainToCache(bucket, object.ContentConnector.String, object.ContentSize.Int64)
		if cipherFile, err = os.Open(cipherFilePathCached); err != nil {
			return &AppError{500, err, "Error opening recently cached file"}
		}
	}

	//Update the timestamps to note the last time it was used
	// This is done here, as well as successful end just in case of failures midstream.
	tm := time.Now()
	os.Chtimes(cipherFilePathCached, tm, tm)

	//We have the file and it's open.  Be sure to close it.
	defer cipherFile.Close()

	w.Header().Set("Content-Type", object.ContentType.String)
	if object.ContentSize.Valid && object.ContentSize.Int64 > int64(0) {
		w.Header().Set("Content-Length", strconv.FormatInt(object.ContentSize.Int64, 10))
		w.Header().Set("Content-Disposition", "inline; filename="+url.QueryEscape(object.Name))
	}
	w.Header().Set("Accept-Ranges", "none")

	//A visibility hack, to reveal metadata about the object in the respone header
	if withMetadata {
		protocolObject := mapping.MapODObjectToObject(object)
		protocolObjectAsJSONBytes, err := json.Marshal(protocolObject)
		if err != nil {
			log.Printf("Unable to marshal object metadata:%v", err)
		} else {
			jsonStringified := string(protocolObjectAsJSONBytes)
			w.Header().Set("Object-Data", jsonStringified)
		}
	}

	//Actually send back the cipherFile
	_, _, err = utils.DoCipherByReaderWriter(
		cipherFile,
		w,
		encryptKey,
		object.EncryptIV,
		"client downloading",
	)

	if err != nil {
		// At this point, we've already started sending data to the client,
		// and so we cant change headers. All we can do is log this error
		// for reference.
		// Reportedly a common cause is Firefox ceasing a stream with a
		// follow up for a range request, which we dont support at this
		// time.
		log.Printf("Error sending decrypted cipherFile (%s): %v", cipherFilePathCached, err)
		// return &AppError{
		// 	Code: 500,
		// 	Err:  err,
		// 	Msg:  fmt.Sprintf("error sending decrypted cipherFile (%s)", cipherFilePathCached),
		// }
	}

	//Update the timestamps to note the last time it was used
	tm = time.Now()
	os.Chtimes(cipherFilePathCached, tm, tm)

	return nil
}

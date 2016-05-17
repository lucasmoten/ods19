package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"net/url"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
)

// getObjectStream gets object data stored in object-drive
func (h AppServer) getObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// req, err := httputil.DumpRequest(r, true)
	// if err != nil {
	// 	log.Printf("unable to dump http request:%v", err)
	// } else {
	// 	log.Printf("%s", string(req))
	// }
	var requestObject models.ODObject
	var err error

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error parsing URI")
		return
	}

	// Retrieve existing object from the data store
	object, err := h.DAO.GetObject(requestObject, true)
	if err != nil {
		sendErrorResponse(&w, 500, err, "Error retrieving object")
		return
	}

	if len(object.ID) == 0 {
		sendErrorResponse(&w, 400, err, "Object retrieved doesn't have an id")
		return
	}

	//Performance count this operation
	beganAt := h.Tracker.BeginTime(performance.DownloadCounter)
	herr := h.getObjectStreamWithObject(ctx, w, r, object)
	transferred := object.ContentSize.Int64
	//Make sure that we count as zero bytes if there was a download error from S3
	if herr != nil {
		transferred = 0
	}
	h.Tracker.EndTime(
		performance.DownloadCounter,
		beganAt,
		performance.SizeJob(transferred),
	)
	//And then return if something went wrong
	if herr != nil {
		sendAppErrorResponse(&w, herr)
		return
	}
	countOKResponse()
}

// getObjectStreamWithObject is the continuation after we retrieved the object from the database
func (h AppServer) getObjectStreamWithObject(ctx context.Context, w http.ResponseWriter, r *http.Request, object models.ODObject) *AppError {

	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return NewAppError(500, err, "Could not determine user")
	}

	// Get the key from the permission
	var permission *models.ODObjectPermission
	var fileKey []byte

	if len(object.Permissions) == 0 {
		return NewAppError(403, fmt.Errorf("We cannot decrypt files lacking permissions"), "Unauthorized")
	}

	if object.IsDeleted {
		switch {
		case object.IsExpunged:
			return NewAppError(410, err, "The object no longer exists.")
		case object.IsAncestorDeleted:
			return NewAppError(405, err, "The object cannot be modified because an ancestor is deleted.")
		default:
			return NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
		}
	}
	//XXX watch for very large number of permissions on a file!
	for _, v := range object.Permissions {
		permission = &v
		if permission.AllowRead && strings.ToLower(permission.Grantee) == caller.DistinguishedName {
			fileKey = permission.EncryptKey
			//Unscramble the fileKey with the masterkey - will need it once more on retrieve
			utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, fileKey)
			// Once we have a match, quit looking and avoid reapplying passphrase
			break
		}
	}
	if len(fileKey) == 0 {
		return NewAppError(403, fmt.Errorf("no applicable permission found"), "Unauthorized")
	}

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &object)
	if err != nil {
		return NewAppError(500, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return NewAppError(403, err, "Unauthorized")
	}

	// Fail fast: Don't even look at cache or retrieve if the file size is 0
	if !object.ContentSize.Valid || object.ContentSize.Int64 <= int64(0) {
		return NewAppError(204, nil, "No content")
	}

	// Database checks and AAC checks take time, particularly AAC.
	// We may have just uploaded this file, with it still draining
	// to S3 in the background.  So open the file as late as possible
	// so that we take advantage of that parallelism.
	h.getAndStreamFile(ctx, &object, w, fileKey, true)

	return nil
}

// drainToCache is the function that retrieves things back out of the drain, and into the cache
// This is a trivial wrapper around the copy that does performance counting.
func drainToCache(
	dp DrainProvider,
	t *performance.JobReporters,
	bucket *string,
	theFile FileId,
	length int64,
) (*AppError, error) {
	beganAt := t.BeginTime(performance.S3DrainFrom)
	herr, err := dp.DrainToCache(bucket, theFile)
	//Be careful to count failed transfers as zero bytes transferred in performance counter
	//I'm assuming that all errors result in a failed file transfer, which is good enough
	//to get close to correct statistics.
	transferred := length
	if herr != nil || err != nil {
		transferred = 0
	}
	t.EndTime(performance.S3DrainFrom, beganAt, performance.SizeJob(transferred))
	if herr != nil {
		return herr, err
	}
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// handleCacheMiss deals with the case where we go to retrieve a file, and we want to
// make a better effort than to throw an exception because it is not cached in our local cache.
// if another routine is caching, then wait for that to finish.
// if nobody is caching it, then we start that process.
func handleCacheMiss(dp DrainProvider, t *performance.JobReporters, object *models.ODObject, cipherFilePathCached FileNameCached) (*os.File, *AppError) {
	var err error

	// We have no choice but to recache and wait.  We can wait for:
	//   - another goroutine that is .caching to finish
	//   - we ourselves start .caching the file
	// It's possible the client will get impatient and not wait (particularly in a browser).

	log.Printf("file is not cached.  Caching it now.")
	bucket := &config.DefaultBucket

	//XXX - the blocking time waiting for this could be very long
	// - it is not guaranteed that the proxy will allow us to just stall for a long time
	// - the user may hit the cancel button in the browser or the app instead of
	//   waiting for it to make it into the cache.
	// - Proxies may just cut the connection if there is no traffic passing.  They can't tell if we are stuck.
	// - so, let's bring it to cache such that it continues if the user disconnects
	// - The user may simply cut the connection and try later.
	//    - Should we put an ETA into the header and send the ETA now?
	//      That way, the user can decide whether to wait or come back later.
	//      But that sends back a 200 OK because we need to commit to OK in order to write
	//      headers.
	//If the http connection gets cut off, this continues to run
	drainingDone := make(chan *AppError, 1)
	alreadyDone := make(chan int, 1)
	defer func() { alreadyDone <- 1 }()

	go func() {
		//If it's caching, then just wait until the file exists.
		// a .caching file should NOT exist if there is not a goroutine actually caching the file.
		// This is why we delete .caching files on startup, and when caching files, we delete .caching
		// file if the goroutine fails.
		rName := FileId(object.ContentConnector.String)
		cachingPath := dp.Resolve(NewFileName(rName, ".caching"))
		cachedPath := dp.Resolve(NewFileName(rName, ".cached"))
		var herr *AppError
		if _, err := dp.Files().Stat(cachingPath); os.IsNotExist(err) {
			//Start caching the file because this is not happening already.
			herr, err = drainToCache(dp, t, bucket, FileId(rName), object.ContentSize.Int64)
			if err != nil || herr != nil {
				//We are not in the http thread.  log this problem though
				log.Printf("!!! unable to drain to cache:%v %v", herr, err)
			}
		} else {
			// Just stall until the cached file exists - somebody else is caching it.
			// Using this simple-minded stalling algorithm, we wait 2x longer or up to 30 seconds longer than necessary.
			sleepTime := time.Duration(1 * time.Second)
			waitMax := time.Duration(30 * time.Second)
			for {
				if _, err := dp.Files().Stat(cachedPath); os.IsNotExist(err) {
					time.Sleep(sleepTime)
					if sleepTime < waitMax {
						sleepTime *= 2
					}
				} else {
					break
				}
			}
		}
		// send back an error code, if we are not already done
		select {
		case _ = <-alreadyDone:
			close(alreadyDone)
		default:
			drainingDone <- herr
		}
		log.Printf("done stalling on %s", cipherFilePathCached)
	}()

	// Wait for the file.  The client might cut off, but we want to keep caching in any case.
	herr := <-drainingDone
	if herr != nil {
		return nil, herr
	}
	close(drainingDone)

	// The file should now exist
	var cipherFile *os.File
	if cipherFile, err = dp.Files().Open(cipherFilePathCached); err != nil {
		return nil, NewAppError(500, err, "Error opening recently cached file")
	}

	return cipherFile, nil
}

// We would like to have a .cached file, but an .uploaded file will do.
func searchForCachedOrUploadedFile(d DrainProvider, cipherFilePathCached, cipherFilePathUploaded FileNameCached) (*os.File, *AppError) {
	var cipherFile *os.File
	var err error
	//Try to find the cached file
	if cipherFile, err = d.Files().Open(cipherFilePathCached); err != nil {
		if os.IsNotExist(err) {
			//Try the file being uploaded into S3
			if cipherFile, err = d.Files().Open(cipherFilePathUploaded); err != nil {
				if os.IsNotExist(err) {
					//Maybe it's cached now?
					if cipherFile, err = d.Files().Open(cipherFilePathCached); err != nil {
						if os.IsNotExist(err) {
							//File is really not cached or uploaded.
							//If it's caching, it's partial anyway
							//leave cipherFile nil, and wait for re-cache
						} else {
							//Some other error.
							return cipherFile, NewAppError(500, err, "Error opening file as cached state")
						}
					} else {
						//cached file exists
					}
				} else {
					//Some other error.
					return cipherFile, NewAppError(500, err, "Error opening file as uploaded state")
				}
			} else {
				//uploaded file exists.  use it.
				log.Printf("using uploaded file")
			}
		} else {
			//Some other error.
			return cipherFile, NewAppError(500, err, "Error opening file as initial cached state")
		}
	} else {
		//the cached file exists
	}
	return cipherFile, nil
}

// This func broken out from the getObjectStream. It still needs refactored to
// be more maintainable and make use of an interface for the content streams
func (h AppServer) getAndStreamFile(ctx context.Context, object *models.ODObject, w http.ResponseWriter, encryptKey []byte, withMetadata bool) *AppError {
	var err error
	var herr *AppError

	//Fall back on the uploaded copy for download if we need to
	var cipherFile *os.File
	d := h.DrainProvider
	rName := FileId(object.ContentConnector.String)
	cipherFilePathCached := d.Resolve(NewFileName(rName, ".cached"))
	cipherFilePathUploaded := d.Resolve(NewFileName(rName, ".uploaded"))

	cipherFile, herr = searchForCachedOrUploadedFile(h.DrainProvider, cipherFilePathCached, cipherFilePathUploaded)
	if herr != nil {
		return herr
	}

	// Check if cipherFile was assigned, denoting whether or not pulling from cache
	if cipherFile == nil {
		cipherFile, herr = handleCacheMiss(h.DrainProvider, h.Tracker, object, cipherFilePathCached)
		if herr != nil {
			return herr
		}
	}

	//Update the timestamps to note the last time it was used
	// This is done here, as well as successful end just in case of failures midstream.
	tm := time.Now()
	d.Files().Chtimes(cipherFilePathCached, tm, tm)

	//We have the file and it's open.  Be sure to close it.
	defer cipherFile.Close()

	//!!! We have committed to 200 OK at this point of setting the header.
	// Any errors that happen after this point will not be seen by the user.

	w.Header().Set("Content-Type", object.ContentType.String)
	if object.ContentSize.Valid && object.ContentSize.Int64 > int64(0) {
		w.Header().Set("Content-Length", strconv.FormatInt(object.ContentSize.Int64, 10))
		w.Header().Set("Content-Disposition", "inline; filename="+url.QueryEscape(object.Name))
	}
	w.Header().Set("Accept-Ranges", "none")

	//Actually send back the cipherFile
	_, _, err = utils.DoCipherByReaderWriter(
		cipherFile,
		w,
		encryptKey,
		object.EncryptIV,
		"client downloading",
	)

	if err != nil {
		//TODO: this needs to be counted somehow
		log.Printf("Error sending decrypted cipherFile (%s): %v", cipherFilePathCached, err)
	}

	//Update the timestamps to note the last time it was used
	tm = time.Now()
	d.Files().Chtimes(cipherFilePathCached, tm, tm)

	return nil
}

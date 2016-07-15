package server

import (
	"crypto/aes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"net/url"

	"encoding/hex"
	"encoding/json"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	db "decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
)

const (
	//B is Bytes unit
	B = int64(1)
	//kB is KBytes unit
	kB = 1024 * B
	//MB is MegaBytes unit
	MB = 1024 * kB
	//GB is Gigabytes unit
	GB = 1024 * MB
)

func extractByteRange(r *http.Request) (*utils.ByteRange, error) {
	var err error
	byteRangeSpec := r.Header.Get("Range")
	parsedByteRange := utils.NewByteRange()
	if len(byteRangeSpec) > 0 {
		typeOfRange := strings.Split(byteRangeSpec, "=")
		if typeOfRange[0] == "bytes" {
			startStop := strings.Split(typeOfRange[1], "-")
			parsedByteRange.Start, err = strconv.ParseInt(startStop[0], 10, 64)
			if err != nil {
				return nil, err
			}
			if len(startStop[1]) > 0 {
				parsedByteRange.Stop, err = strconv.ParseInt(startStop[1], 10, 64)
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, fmt.Errorf("could not understand range type")
		}
	} else {
		//If there is no byte range asked for, then don't return one.  It's important, because
		// if client is not asking for a byte range, then he's expecting a 200.
		return nil, nil
	}
	return parsedByteRange, err
}

// getObjectStream gets object data stored in object-drive
func (h AppServer) getObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	var err error
	var requestObject models.ODObject

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		return NewAppError(500, err, "Error parsing URI")
	}
	dao := DAOFromContext(ctx)

	// Retrieve existing object from the data store
	object, err := dao.GetObject(requestObject, true)
	if err != nil {
		if err == db.ErrNoRows {
			return NewAppError(404, err, "Not found")
		}
		return NewAppError(500, err, "Error retrieving object")
	}

	if len(object.ID) == 0 {
		return NewAppError(400, err, "Object retrieved doesn't have an id")
	}

	//Performance count this operation
	beganAt := h.Tracker.BeginTime(performance.DownloadCounter)
	transferred, herr := h.getObjectStreamWithObject(ctx, w, r, object)
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
		return herr
	}
	return nil
}

// getObjectStreamWithObject is the continuation after we retrieved the object from the database
// returns the actual bytes transferred due to range requesting
func (h AppServer) getObjectStreamWithObject(ctx context.Context, w http.ResponseWriter, r *http.Request, object models.ODObject) (int64, *AppError) {

	var err error

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return 0, NewAppError(500, err, "Could not determine user")
	}

	// Get the key from the permission
	var permission *models.ODObjectPermission
	var fileKey []byte

	if len(object.Permissions) == 0 {
		return 0, NewAppError(403, fmt.Errorf("We cannot decrypt files lacking permissions"), "Unauthorized")
	}

	if object.IsDeleted {
		switch {
		case object.IsExpunged:
			return 0, NewAppError(410, err, "The object no longer exists.")
		case object.IsAncestorDeleted:
			return 0, NewAppError(405, err, "The object cannot be modified because an ancestor is deleted.")
		default:
			return 0, NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
		}
	}
	//XXX watch for very large number of permissions on a file!
	for _, v := range object.Permissions {
		permission = &v
		if permission.AllowRead && strings.ToLower(permission.Grantee) == caller.DistinguishedName {
			if models.EqualsPermissionMAC(h.MasterKey, permission) == false {
				//Check the integrity of the grant before giving a stream
				return 0, NewAppError(403, nil, "Unauthorized")
			}
			fileKey = utils.ApplyPassphrase(h.MasterKey, permission.PermissionIV, permission.EncryptKey)
			break
		}
	}
	if len(fileKey) == 0 {
		return 0, NewAppError(403, fmt.Errorf("no applicable permission found"), "Unauthorized")
	}

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	hasAACAccess, err := h.isUserAllowedForObjectACM(ctx, &object)
	if err != nil {
		return 0, NewAppError(502, err, "Error communicating with authorization service")
	}
	if !hasAACAccess {
		return 0, NewAppError(403, err, "Unauthorized")
	}

	if !object.ContentSize.Valid || object.ContentSize.Int64 <= int64(0) {
		return 0, NewAppError(204, nil, "No content", zap.Int64("bytes", object.ContentSize.Int64))
	}

	contentLength, herr := h.getAndStreamFile(ctx, &object, w, r, fileKey, true)

	return contentLength, herr
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

// backgroundRecache deals with the case where we go to retrieve a file, and we want to
// make a better effort than to throw an exception because it is not cached in our local cache.
// if another routine is caching, then wait for that to finish.
// if nobody is caching it, then we start that process.
func backgroundRecache(ctx context.Context, d DrainProvider, t *performance.JobReporters, object *models.ODObject, cipherFilePathCached FileNameCached) {
	//var err error
	logger := LoggerFromContext(ctx)
	// We have no choice but to recache and wait.  We can wait for:
	//   - another goroutine that is .caching to finish
	//   - we ourselves start .caching the file
	// It's possible the client will get impatient and not wait (particularly in a browser).
	logger.Info(
		"caching file",
		zap.String("filename", d.Files().Resolve(cipherFilePathCached)),
	)
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
	go func() {
		//If it's caching, then just wait until the file exists.
		// a .caching file should NOT exist if there is not a goroutine actually caching the file.
		// This is why we delete .caching files on startup, and when caching files, we delete .caching
		// file if the goroutine fails.
		rName := FileId(object.ContentConnector.String)
		cachingPath := d.Resolve(NewFileName(rName, ".caching"))
		cachedPath := d.Resolve(NewFileName(rName, ".cached"))
		var herr *AppError
		if _, err := d.Files().Stat(cachingPath); os.IsNotExist(err) {
			//Start caching the file because this is not happening already.
			herr, err = drainToCache(d, t, bucket, FileId(rName), object.ContentSize.Int64)
			if err != nil || herr != nil {
				var errStr string
				if err != nil {
					errStr = err.Error()
				}
				var herrStr string
				if herr != nil {
					herrStr = herr.Error.Error()
				}
				logger.Error(
					"unable to drain cache",
					zap.String("err", errStr),
					zap.String("herr", herrStr),
				)
			}
		} else {
			// Just stall until the cached file exists - somebody else is caching it.
			// Using this simple-minded stalling algorithm, we wait 2x longer or up to 30 seconds longer than necessary.
			sleepTime := time.Duration(1 * time.Second)
			waitMax := time.Duration(30 * time.Second)
			for {
				if _, err := d.Files().Stat(cachedPath); os.IsNotExist(err) {
					time.Sleep(sleepTime)
					if sleepTime < waitMax {
						sleepTime *= 2
					}
				} else {
					break
				}
			}
		}
		logger.Info(
			"stall done",
			zap.String("filename", d.Files().Resolve(cipherFilePathCached)),
		)
	}()
}

// We would like to have a .cached file, but an .uploaded file will do.
func useLocalFile(d DrainProvider, cipherFilePathCached, cipherFilePathUploaded FileNameCached, cipherStartAt int64) (*os.File, *AppError) {
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
				//Use uploaded file
			}
		} else {
			//Some other error.
			return cipherFile, NewAppError(500, err, "Error opening file as initial cached state")
		}
	} else {
		//the cached file exists
	}

	//Seek to where we should start reading the cipher
	if cipherFile != nil {
		_, err = cipherFile.Seek(cipherStartAt, 0)
		if err != nil {
			cipherFile.Close()
			fName := d.Files().Resolve(cipherFilePathCached)
			return cipherFile, NewAppError(500, err,
				"Could not seek file",
				zap.String("filename", fName),
				zap.Int64("index", cipherStartAt),
			)
		}
		//Update the timestamps to note the last time it was used
		// This is done here, as well as successful end just in case of failures midstream.
		tm := time.Now()
		d.Files().Chtimes(cipherFilePathCached, tm, tm)
	}

	return cipherFile, nil
}

func adjustIV(originalIV []byte, byteRange *utils.ByteRange) []byte {
	//Skip over blocks we won't use
	iv := originalIV
	if byteRange != nil {

		//Add blocksToSkip to the iv
		//First duplicate the IV
		blocksSkipped := (byteRange.Start / aes.BlockSize)
		blocksToSkip := blocksSkipped
		iv = make([]byte, aes.BlockSize)
		for i := 0; i < aes.BlockSize; i++ {
			iv[i] = originalIV[i]
		}
		//Add blocksToSkip to it (bigEndian)
		var i = 0
		for blocksToSkip > 0 {
			v := uint8(blocksToSkip % 256)
			newV := uint32(v) + uint32(iv[aes.BlockSize-i-1])
			//do add
			iv[aes.BlockSize-i-1] = uint8(newV)
			//do carry
			iv[aes.BlockSize-i-2] += uint8(newV >> 8)
			blocksToSkip >>= 8
			i++
		}
		//Adjust the byte range to match what the file handle skipped already
		byteRange.Start -= blocksSkipped * aes.BlockSize
		if byteRange.Stop != -1 {
			byteRange.Stop -= blocksSkipped * aes.BlockSize
		}
	}
	return iv
}

// The interface for these files is now a valid io.ReadCloser.
// In the case of a cache miss, we no longer wait for the entire file.
// We have an io.ReadCloser() that will fill the bytes by range requesting
// out of S3.  It will not write these bytes to disk in an intermediate step.
func (h AppServer) getAndStreamFile(ctx context.Context, object *models.ODObject, w http.ResponseWriter, r *http.Request, encryptKey []byte, withMetadata bool) (int64, *AppError) {
	var err error
	var herr *AppError
	logger := LoggerFromContext(ctx)

	//Prepare for range requesting
	byteRange, err := extractByteRange(r)
	if err != nil {
		return 0, NewAppError(400, err, "Unable to parse byte range")
	}
	var blocksSkipped int64
	var cipherStartAt int64
	if byteRange != nil {
		blocksSkipped = (byteRange.Start / aes.BlockSize)
		cipherStartAt = blocksSkipped * aes.BlockSize
	}

	//We should have classification banner with the content, as
	//the banner is at least as mandatory as the filename.
	//If you resolve a link, you otherwise would not know the classification
	//without stopping and downloading the object metadata first.
	//This may also be useful for the client and proxies with respect to caching
	//as well.
	if object.RawAcm.Valid {
		var acm interface{}
		acmBytes := []byte(object.RawAcm.String)
		err := json.Unmarshal(acmBytes, &acm)
		if err == nil {
			acmData, acmDataOk := acm.(map[string]interface{})
			if acmDataOk {
				banner, bannerOk := acmData["banner"].(string)
				if bannerOk {
					w.Header().Set("Classification-Banner", banner)
				}
			}
		} else {
			logger.Warn(
				"acm parse",
				zap.String("err", err.Error()),
				zap.String("acm", object.RawAcm.String),
			)
		}
	}

	//When setting headers, take measures to handle byte range requesting
	w.Header().Set("Content-Type", object.ContentType.String)
	var start = int64(0)
	var stop = object.ContentSize.Int64 - 1
	var fullLength = object.ContentSize.Int64
	if object.ContentSize.Valid {
		var reportedContentLength = fullLength

		w.Header().Set("Accept-Ranges", "bytes")

		if byteRange != nil {
			start = byteRange.Start
			if byteRange.Stop != -1 {
				stop = byteRange.Stop
			}
			reportedContentLength = stop + 1 - start
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", reportedContentLength))
		w.Header().Set("Content-Disposition", "inline; filename="+url.QueryEscape(object.Name))
	}

	clientEtag := r.Header.Get("If-None-Match")
	if object.ContentSize.Valid {
		contentHash := hex.EncodeToString(object.ContentHash)
		if byteRange != nil {
			rangeResponse := fmt.Sprintf("bytes %d-%d/%d", start, stop, fullLength)
			w.Header().Set("Content-Range", rangeResponse)
			etag := fmt.Sprintf("\"%s\"", contentHash)
			w.Header().Set("Etag", etag)
			if clientEtag == etag {
				return 0, NewAppError(http.StatusNotModified, nil, "Not Modified")
			}
			//Note that if we return a nil error, the stats collector will think we got a 200
			//Begin writing a 206... one of the rare codes that still returns content
			w.WriteHeader(http.StatusPartialContent)
		} else {
			etag := fmt.Sprintf("\"%s\"", contentHash)
			w.Header().Set("Etag", etag)
			if clientEtag == etag {
				return 0, NewAppError(http.StatusNotModified, nil, "Not Modified")
			}
			//Begin writing back a normal 200
			w.WriteHeader(http.StatusOK)
		}
	}

	//Fall back on the uploaded copy for download if we need to
	var cipherFile *os.File
	d := h.DrainProvider
	rName := FileId(object.ContentConnector.String)
	cipherFilePathCached := d.Resolve(NewFileName(rName, ".cached"))
	cipherFilePathUploaded := d.Resolve(NewFileName(rName, ".uploaded"))
	cipherFile, herr = useLocalFile(h.DrainProvider, cipherFilePathCached, cipherFilePathUploaded, cipherStartAt)
	if herr != nil {
		return 0, herr
	}

	//Was it found already?
	var cipherReader io.ReadCloser
	if cipherFile == nil {
		//Not found, so pull the file to disk from S3 in the background,
		backgroundRecache(ctx, h.DrainProvider, h.Tracker, object, cipherFilePathCached)
		totalLength := object.ContentSize.Int64
		//...where this looks like a normal io.ReadCloser, but it keeps range requesting to
		//...keep full with bytes, as requested.  This deals neatly with clients that connect,
		//...and only pull a small number of bytes.
		cipherReader, err = d.NewS3Puller(logger, rName, totalLength, cipherStartAt, -1)
		if err != nil {
			return 0, NewAppError(500, err, "s3 pull fail", zap.String("err", err.Error()))
		}
	} else {
		cipherReader = cipherFile
	}

	defer cipherReader.Close()

	//Skip over blocks we won't use, and adjust byteRange to match it
	iv := adjustIV(object.EncryptIV, byteRange)
	//Actually send back the cipherFile
	var actualLength int64
	_, actualLength, err = utils.DoCipherByReaderWriter(
		logger,
		cipherReader,
		w,
		encryptKey,
		iv,
		"client downloading",
		byteRange,
	)

	logger.Info("transaction down", zap.Int64("bytes", actualLength))
	if err != nil {
		//Error here isn't a constant, but it's indicative of client disconnecting and
		//not bothering to eat all the bytes we sent back (as promised).  So be quiet
		//in the case of broken pipe.
		if strings.Contains(err.Error(), "broken pipe") ||
			strings.Contains(err.Error(), "connection reset by peer") {
			//Clients are allowed to disconnect and not accept all bytes we are sending back
		} else {
			logger.Error(
				"client disconnect",
				zap.String("filename", d.Files().Resolve(cipherFilePathCached)),
				zap.String("err", err.Error()),
			)
		}
	}
	if object.ContentSize.Valid && byteRange != nil {
		return actualLength, NewAppError(http.StatusPartialContent, nil, "Partial Content")
	} else {
		return actualLength, nil
	}
}

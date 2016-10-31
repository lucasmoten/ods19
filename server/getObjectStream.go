package server

import (
	"crypto/aes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"net/url"

	"encoding/hex"
	"encoding/json"

	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/crypto"

	db "decipher.com/object-drive-server/dao"
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

func extractByteRange(r *http.Request) (*crypto.ByteRange, error) {
	var err error
	byteRangeSpec := r.Header.Get("Range")
	parsedByteRange := crypto.NewByteRange()
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
		//Equality issue on errors?
		if err.Error() == db.ErrNoRows.Error() {
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

	var NoBytesReturned int64
	var err error

	// // Get caller value from ctx.
	// caller, ok := CallerFromContext(ctx)
	// if !ok {
	// 	return NoBytesReturned, NewAppError(500, err, "Could not determine user")
	// }

	if object.IsDeleted {
		switch {
		case object.IsExpunged:
			return NoBytesReturned, NewAppError(410, err, "The object no longer exists.")
		case object.IsAncestorDeleted:
			return NoBytesReturned, NewAppError(405, err, "The object cannot be modified because an ancestor is deleted.")
		default:
			return NoBytesReturned, NewAppError(405, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
		}
	}

	// Check read permission, and capture permission for the encryptKey
	// Check if the user has permissions to read the ODObject
	//		Permission.grantee matches caller, and AllowRead is true
	ok, userPermission := isUserAllowedToReadWithPermission(ctx, h.MasterKey, &object)
	if !ok {
		return NoBytesReturned, NewAppError(403, errors.New("Forbidden"), "Forbidden - User does not have permission to read/view this object")
	}
	// Using captured permission, derive filekey
	var fileKey []byte
	fileKey = crypto.ApplyPassphrase(h.MasterKey, userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		return NoBytesReturned, NewAppError(500, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object")
	}

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	if err = h.isUserAllowedForObjectACM(ctx, &object); err != nil {
		return NoBytesReturned, ClassifyObjectACMError(err)
	}

	if !object.ContentSize.Valid || object.ContentSize.Int64 <= int64(0) {
		return NoBytesReturned, NewAppError(204, nil, "No content", zap.Int64("bytes", object.ContentSize.Int64))
	}

	disposition := "inline"
	overrideDisposition := r.URL.Query().Get("disposition")
	if len(overrideDisposition) > 0 {
		disposition = overrideDisposition
	}
	contentLength, herr := h.getAndStreamFile(ctx, &object, w, r, fileKey, true, disposition)

	return contentLength, herr
}

func adjustIV(originalIV []byte, byteRange *crypto.ByteRange) []byte {
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

//Get a banner or a portion from here
func acmExtractItem(item string, rawAcm string) (string, error) {
	var acm interface{}
	var err error
	acmBytes := []byte(rawAcm)
	err = json.Unmarshal(acmBytes, &acm)
	if err == nil {
		acmData, acmDataOk := acm.(map[string]interface{})
		if acmDataOk {
			val, ok := acmData[item].(string)
			if ok {
				return val, nil
			}
		}
	}
	return "", err
}

// The interface for these files is now a valid io.ReadCloser.
// In the case of a cache miss, we no longer wait for the entire file.
// We have an io.ReadCloser() that will fill the bytes by range requesting
// out of S3.  It will not write these bytes to disk in an intermediate step.
func (h AppServer) getAndStreamFile(ctx context.Context, object *models.ODObject, w http.ResponseWriter, r *http.Request, encryptKey []byte, withMetadata bool, disposition string) (int64, *AppError) {
	var err error
	var finalStatus *AppError
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
		if object.RawAcm.Valid {
			banner, err := acmExtractItem("banner", object.RawAcm.String)
			if err != nil {
				logger.Warn(
					"acm parse",
					zap.String("err", err.Error()),
					zap.String("acm", object.RawAcm.String),
				)
			} else {
				w.Header().Set("Classification-Banner", banner)
			}
		}
	}

	//When setting headers, take measures to handle byte range requesting
	w.Header().Set("Content-Type", object.ContentType.String)
	var start = int64(0)
	var stop = object.ContentSize.Int64 - 1
	var fullLength = object.ContentSize.Int64
	clientEtag := r.Header.Get("If-None-Match")
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
		//typical disposition values: inline, attachment - RFC2183.
		w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, url.QueryEscape(object.Name)))
		//RFC2183 talks about Content-Description.  We should set this.
		if object.Description.Valid && len(object.Description.String) > 0 {
			w.Header().Set("Content-Description", object.Description.String)
		}
		//This contentHash is a sha256 of the full plaintext.
		contentHash := hex.EncodeToString(object.ContentHash)
		if byteRange != nil {
			rangeResponse := fmt.Sprintf("bytes %d-%d/%d", start, stop, fullLength)
			w.Header().Set("Content-Range", rangeResponse)
			etag := fmt.Sprintf("\"%s\"", contentHash)
			w.Header().Set("Etag", etag)
			if clientEtag == etag {
				w.Header().Del("Content-Length")
				return 0, NewAppError(http.StatusNotModified, nil, "Not Modified")
			}
			//Note that if we return a nil error, the stats collector will think we got a 200
			//Begin writing a 206... one of the rare codes that still returns content
			w.WriteHeader(http.StatusPartialContent)
			//We cant return yet because we need to send bytes back, but we should return 206 in the end.
			finalStatus = NewAppError(http.StatusPartialContent, nil, "Partial Content")
		} else {
			etag := fmt.Sprintf("\"%s\"", contentHash)
			w.Header().Set("Etag", etag)
			if clientEtag == etag {
				w.Header().Del("Content-Length")
				return 0, NewAppError(http.StatusNotModified, nil, "Not Modified")
			}
			//Begin writing back a normal 200
			w.WriteHeader(http.StatusOK)
		}
	}

	d := ciphertext.FindCiphertextCacheByObject(object)
	rName := ciphertext.FileId(object.ContentConnector.String)
	cipherFilePathCached := d.Resolve(ciphertext.NewFileName(rName, ".cached"))
	totalLength := object.ContentSize.Int64
	isLocalPuller := false
	var cipherReader io.ReadCloser

	//Pull the file from the cache
	cipherReader, isLocalPuller, err = d.NewPuller(logger, rName, totalLength, cipherStartAt, -1)
	if err != nil {
		return 0, NewAppError(500, err, err.Error())
	}
	defer cipherReader.Close()
	//If it didn't come from the files in the cache, then make sure we ReCache it for future use
	//TODO: it might be desirable to only do this probabilistically
	if !isLocalPuller {
		go func() {
			beganAt := h.Tracker.BeginTime(performance.S3DrainFrom)
			d.BackgroundRecache(rName, object.ContentSize.Int64)
			h.Tracker.EndTime(performance.S3DrainFrom, beganAt, performance.SizeJob(fullLength))
		}()
	}

	//Skip over blocks we won't use, and adjust byteRange to match it
	iv := adjustIV(object.EncryptIV, byteRange)
	//Actually send back the cipherFile
	var actualLength int64
	_, actualLength, err = crypto.DoCipherByReaderWriter(
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
	//the finalStatus is not necessarily an error, but requires a status code, because nil implies 200.
	return actualLength, finalStatus
}

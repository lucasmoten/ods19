package server

import (
	"crypto/aes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"golang.org/x/net/context"

	"encoding/hex"
	"encoding/json"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/ciphertext"
	"bitbucket.di2e.net/dime/object-drive-server/crypto"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"

	db "bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/performance"
)

func extractByteRange(r *http.Request) (*crypto.ByteRange, error) {
	var err error
	byteRangeSpec := r.Header.Get("Range")
	if len(byteRangeSpec) == 0 {
		// If there is no byte range requested, then don't return one.
		return nil, nil
	}
	parsedByteRange := crypto.NewByteRange()
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
	return parsedByteRange, err
}

// getObjectStream gets object data stored in object-drive
func (h AppServer) getObjectStream(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	var err error
	var requestObject models.ODObject
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	requestObject, err = parseGetObjectRequest(ctx)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Error parsing URI")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.ObjectID = hex.EncodeToString(requestObject.ID)
	gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(requestObject.ID))

	dao := DAOFromContext(ctx)

	// Retrieve existing object from the data store
	dbObject, err := dao.GetObject(requestObject, true)
	if err != nil {
		//Equality issue on errors?
		if err.Error() == db.ErrNoRows.Error() {
			herr := NewAppError(http.StatusNotFound, err, "Not found")
			h.publishError(gem, herr)
			return herr
		}
		herr := NewAppError(http.StatusInternalServerError, err, "Error retrieving object")
		h.publishError(gem, herr)
		return herr
	}
	gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(dbObject))
	gem.Payload.ChangeToken = dbObject.ChangeToken

	if len(dbObject.ID) == 0 {
		herr := NewAppError(http.StatusInternalServerError, err, "Object retrieved doesn't have an id")
		h.publishError(gem, herr)
		return herr
	}

	//Performance count this operation
	ctx = ContextWithGEM(ctx, gem)
	trackingEnabled := false
	var beganAt performance.BeganJob
	if trackingEnabled {
		beganAt = h.Tracker.BeginTime(performance.DownloadCounter)
	}
	transferred, herr := h.getObjectStreamWithObject(ctx, w, r, dbObject)
	if trackingEnabled {
		//Make sure that we count as zero bytes if there was a download error from S3
		if herr != nil {
			transferred = 0
		}
		h.Tracker.EndTime(
			performance.DownloadCounter,
			beganAt,
			performance.SizeJob(transferred),
		)
	}
	//And then return if something went wrong
	if herr != nil {
		if herr.Error != nil {
			h.publishError(gem, herr)
		} else {
			h.publishSuccess(gem, w)
		}
		return herr
	}
	h.publishSuccess(gem, w)
	return nil
}

// getObjectStreamWithObject is the continuation after we retrieved the object from the database
// returns the actual bytes transferred due to range requesting
func (h AppServer) getObjectStreamWithObject(ctx context.Context, w http.ResponseWriter, r *http.Request, dbObject models.ODObject) (int64, *AppError) {
	caller, _ := CallerFromContext(ctx)
	var NoBytesReturned int64
	var err error
	gem, _ := GEMFromContext(ctx)
	logger := LoggerFromContext(ctx)

	if dbObject.IsDeleted {
		var herr *AppError
		switch {
		case dbObject.IsExpunged:
			herr = NewAppError(http.StatusGone, err, "The object no longer exists")
		case dbObject.IsAncestorDeleted:
			herr = NewAppError(http.StatusConflict, err, "The object cannot be retrieved because an ancestor is deleted.")
		default:
			herr = NewAppError(http.StatusConflict, err, "The object is currently in the trash. Use removeObjectFromtrash to restore it before updating it.")
		}
		h.publishError(gem, herr)
		return NoBytesReturned, herr
	}

	// Check read permission, and capture permission for the encryptKey
	// Check if the user has permissions to read the ODObject
	//		Permission.grantee matches caller, and AllowRead is true
	ok, userPermission := isUserAllowedToReadWithPermission(ctx, &dbObject)
	if !ok {
		herr := NewAppError(http.StatusForbidden, errors.New("Forbidden"), "Forbidden - User does not have permission to read/view this object")
		h.publishError(gem, herr)
		return NoBytesReturned, herr
	}

	dp := ciphertext.FindCiphertextCacheByObject(&dbObject)
	masterKey := dp.GetMasterKey()

	// Using captured permission, derive filekey
	var fileKey []byte
	fileKey = crypto.ApplyPassphrase(masterKey, userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		herr := NewAppError(http.StatusInternalServerError, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object")
		h.publishError(gem, herr)
		return NoBytesReturned, herr
	}

	// Check AAC to compare user clearance to  metadata Classifications
	// 		Check if Classification is allowed for this User
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		herr := NewAppError(authHTTPErr(err), err, err.Error())
		h.publishError(gem, herr)
		return NoBytesReturned, herr
	}

	if !dbObject.ContentSize.Valid || dbObject.ContentSize.Int64 <= int64(0) {
		herr := NewAppError(http.StatusNoContent, nil, "No content")
		h.publishSuccess(gem, w)
		return NoBytesReturned, herr
	}

	disposition := "inline"
	overrideDisposition := r.URL.Query().Get("disposition")
	if len(overrideDisposition) > 0 {
		disposition = overrideDisposition
	}
	contentLength, herr := h.getAndStreamFile(ctx, &dbObject, w, r, fileKey, true, disposition)

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

func sanitizeDisposition(s string) string {
	switch strings.ToLower(s) {
	case "attachment":
		return "attachment"
	case "inline":
		return "inline"
	default:
		return "inline"
	}
}

func sanitizeAgainstCRLFInHeader(s string) string {
	o := s
	// Disallow any carriage returns, line feeds, null
	o = strings.Replace(o, "\r", "", -1)
	o = strings.Replace(o, "%0d", "", -1)
	o = strings.Replace(o, "%0D", "", -1)
	o = strings.Replace(o, "\n", "", -1)
	o = strings.Replace(o, "%0a", "", -1)
	o = strings.Replace(o, "%0A", "", -1)
	o = strings.Replace(o, "%00", "_", -1)
	o = strings.Replace(o, "\x00", "_", -1)
	return o
}

func sanitizeFilename(s string) string {
	o := s
	// Get rid of any leading or trailing whitespace (CR, LF, TAB, SPACE, BACKSPACE, etc)
	o = strings.TrimSpace(o)
	// Get rid of any CRLF headers that support HTTP Header Injection
	o = sanitizeAgainstCRLFInHeader(o)
	// Take only the base part (ie, if name is "something/../../evil/file.txt", then just return the file.txt
	o = path.Base(o)
	o = strings.TrimSpace(o)

	return o
}

// The interface for these files is now a valid io.ReadCloser.
// In the case of a cache miss, we no longer wait for the entire file.
// We have an io.ReadCloser() that will fill the bytes by range requesting
// out of S3.  It will not write these bytes to disk in an intermediate step.
func (h AppServer) getAndStreamFile(ctx context.Context, object *models.ODObject, w http.ResponseWriter, r *http.Request, encryptKey []byte, withMetadata bool, disposition string) (int64, *AppError) {
	var NoBytesReturned int64
	var err error
	gem, _ := GEMFromContext(ctx)
	var finalStatus *AppError
	logger := LoggerFromContext(ctx)

	//Prepare for range requesting
	byteRange, err := extractByteRange(r)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "Unable to parse byte range")
		h.publishError(gem, herr)
		return NoBytesReturned, herr
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
	if h.Conf.HeaderBannerEnabled {
		if object.RawAcm.Valid {
			if object.RawAcm.Valid {
				banner, err := acmExtractItem("banner", object.RawAcm.String)
				if err != nil {
					logger.Warn(
						"acm parse",
						zap.Error(err),
						zap.String("acm", object.RawAcm.String),
					)
				} else {
					w.Header().Set(h.Conf.HeaderBannerName, sanitizeAgainstCRLFInHeader(banner))
				}
			}
		}
	}

	//When setting headers, take measures to handle byte range requesting
	w.Header().Set("Content-Type", sanitizeAgainstCRLFInHeader(object.ContentType.String))
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
		w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, sanitizeDisposition(disposition), sanitizeFilename(object.Name)))
		//RFC2183 talks about Content-Description.  We should set this.
		if object.Description.Valid && len(object.Description.String) > 0 {
			w.Header().Set("Content-Description", sanitizeAgainstCRLFInHeader(object.Description.String))
		}
		//This contentHash is a sha256 of the full plaintext.
		contentHash := hex.EncodeToString(object.ContentHash)
		if byteRange != nil {
			rangeResponse := fmt.Sprintf("bytes %d-%d/%d", start, stop, fullLength)
			w.Header().Set("Content-Range", rangeResponse)
			etag := fmt.Sprintf("\"%s\"", contentHash)
			w.Header().Set("ETag", etag)
			if clientEtag == etag {
				w.Header().Del("Content-Length")
				herr := NewAppError(http.StatusNotModified, nil, "Not Modified")
				return NoBytesReturned, herr
			}
			//Note that if we return a nil error, the stats collector will think we got a 200
			//Begin writing a 206... one of the rare codes that still returns content
			w.WriteHeader(http.StatusPartialContent)
			//We cant return yet because we need to send bytes back, but we should return 206 in the end.
			finalStatus = NewAppError(http.StatusPartialContent, nil, "Partial Content")
		} else {
			etag := fmt.Sprintf("\"%s\"", contentHash)
			w.Header().Set("ETag", etag)
			if clientEtag == etag {
				w.Header().Del("Content-Length")
				herr := NewAppError(http.StatusNotModified, nil, "Not Modified")
				return NoBytesReturned, herr
			}
			//Begin writing back a normal 200
			w.WriteHeader(http.StatusOK)
		}
	}

	logger.Debug("cipher file being resolved", zap.String("id", hex.EncodeToString(object.ID)), zap.String("contentConnector", object.ContentConnector.String))
	d := ciphertext.FindCiphertextCacheByObject(object)
	rName := ciphertext.FileId(object.ContentConnector.String)
	cipherFilePathCached := d.Resolve(ciphertext.NewFileName(rName, ciphertext.FileStateCached))
	totalLength := object.ContentSize.Int64
	isLocalPuller := false
	var cipherReader io.ReadCloser

	//Pull the file from the cache
	logger.Debug("cipher file being pulled", zap.String("contentConnector", object.ContentConnector.String))
	cipherReader, isLocalPuller, err = d.NewPuller(logger, rName, totalLength, cipherStartAt, -1)
	// Ensure reader is closed even for errors as part of fix for DIMEODS-1262
	if cipherReader != nil {
		defer cipherReader.Close()
	}
	if err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, err.Error())
		h.publishError(gem, herr)
		return NoBytesReturned, herr
	}
	//If it didn't come from the files in the cache, then make sure we ReCache it for future use
	//TODO: it might be desirable to only do this probabilistically
	if !isLocalPuller {
		logger.Debug("cipher file was not local will do background recache")
		go func() {
			trackingEnabled := false
			var beganAt performance.BeganJob
			if trackingEnabled {
				beganAt = h.Tracker.BeginTime(performance.S3DrainFrom)
			}
			d.BackgroundRecache(rName, object.ContentSize.Int64)
			if trackingEnabled {
				h.Tracker.EndTime(performance.S3DrainFrom, beganAt, performance.SizeJob(fullLength))
			}
		}()
	}

	//Skip over blocks we won't use, and adjust byteRange to match it
	iv := adjustIV(object.EncryptIV, byteRange)
	//Actually send back the cipherFile
	var actualLength int64
	_, actualLength, err = h.Conf.EncryptableFunctions.DoCipherByReaderWriter(
		logger,
		cipherReader,
		w,
		encryptKey,
		iv,
		"client downloading",
		byteRange,
	)

	logger.Debug("sent content stream to client", zap.Int64("bytes", actualLength), zap.Any("byterange", byteRange))
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
				zap.Error(err),
			)
		}
	}
	//the finalStatus is not necessarily an error, but requires a status code, because nil implies 200.
	return actualLength, finalStatus
}

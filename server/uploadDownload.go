package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"path"
	"strings"
	"time"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/crypto"
	"golang.org/x/net/context"

	"decipher.com/object-drive-server/mapping"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/protocol"
)

//If we are returning potentially after the object has been uploaded to disk,
//then there is a time-span where abort must cleanup after itself
func abortUploadObject(
	logger zap.Logger,
	dp ciphertext.CiphertextCache,
	obj *models.ODObject,
	isMultipart bool,
	herr *AppError) *AppError {
	if isMultipart {
		removeOrphanedFile(logger, dp, obj.ContentConnector.String)
	}
	return herr
}

// This is split because we are going to need to be able to do things in between getting metadata and accepting the bytes
// when we have a masterkey per cache
func (h AppServer) acceptObjectUpload(ctx context.Context, multipartReader *multipart.Reader, obj *models.ODObject,
	grant *models.ODObjectPermission, asCreate bool, afterMeta func(*models.ODObject)) (func(), *AppError) {

	// Get the first part
	part, err := multipartReader.NextPart()
	if err != nil {
		return nil, NewAppError(400, err, "error getting metadata part")
	}

	parsedMetadata, herr := h.acceptObjectUploadMeta(ctx, part, obj, grant, asCreate)
	if herr != nil {
		return nil, herr
	}

	// Get the second part if the first was consumed.
	if parsedMetadata {
		part, err = multipartReader.NextPart()
		if err == io.EOF {
			return nil, NewAppError(400, err, "error getting stream part")
		}
	}

	//This is code inserted in between metadata parse and accepting the stream
	if afterMeta != nil {
		afterMeta(obj)
	}

	// Process the stream
	return h.acceptObjectUploadStream(ctx, part, obj, grant, asCreate, parsedMetadata)
}

// Get an update obj from the caller - we are not persisting anything yet
func (h AppServer) acceptObjectUploadMeta(ctx context.Context, part *multipart.Part, obj *models.ODObject,
	grant *models.ODObjectPermission, asCreate bool) (bool, *AppError) {
	var herr *AppError

	parsedMetadata := false
	var createObjectRequest protocol.CreateObjectRequest
	var updateObjectRequest protocol.UpdateObjectAndStreamRequest

	if part.FormName() == "ObjectMetadata" {
		parsedMetadata = true

		limit := 5 << (10 * 2)
		metadata, err := ioutil.ReadAll(io.LimitReader(part, int64(limit)))
		if err != nil {
			return parsedMetadata, NewAppError(400, err, "could not read json metadata")
		}
		// Parse into a useable struct
		if asCreate {
			err = json.Unmarshal(metadata, &createObjectRequest)
		} else {
			err = json.Unmarshal(metadata, &updateObjectRequest)
		}
		if err != nil {
			return parsedMetadata, NewAppError(400, err, fmt.Sprintf("Could not decode ObjectMetadata: %s", metadata))
		}

		// Validation & Mapping for Create
		if asCreate {
			// Mapping to object
			err = mapping.OverwriteODObjectWithCreateObjectRequest(obj, &createObjectRequest)
			if err != nil {
				return parsedMetadata, NewAppError(400, err, "Could not extract data from json response")
			}
			// Post mapping rules applied for create (not deleted, enforce owner cruds, assign meta)
			if herr := handleCreatePrerequisites(ctx, h, obj); herr != nil {
				return parsedMetadata, herr
			}
		}

		// Validation & Mapping for Update
		if !asCreate {
			// ID in json must match that on the URI
			herr = compareIDFromJSONWithURI(ctx, updateObjectRequest)
			if herr != nil {
				return parsedMetadata, herr
			}
			// ChangeToken must be provided and match the object
			if obj.ChangeToken != updateObjectRequest.ChangeToken {
				return parsedMetadata, NewAppError(400, nil, "Changetoken must be up to date")
			}
			// Mapping to object
			err = mapping.OverwriteODObjectWithUpdateObjectAndStreamRequest(obj, &updateObjectRequest)
			if err != nil {
				return parsedMetadata, NewAppError(400, err, "Could not extract data from json response")
			}
		}

		// Whether creating or updating, the ACM must have a value
		if len(obj.RawAcm.String) == 0 {
			return parsedMetadata, NewAppError(400, err, "An ACM must be specified")
		}
	}
	return parsedMetadata, nil
}

// Get the bytes from the caller.
func (h AppServer) acceptObjectUploadStream(ctx context.Context, part *multipart.Part, obj *models.ODObject,
	grant *models.ODObjectPermission, asCreate bool, parsedMetadata bool) (func(), *AppError) {

	var herr *AppError
	var err error
	var drainFunc func()

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return nil, NewAppError(400, fmt.Errorf("User not provided in context"), "Could not determine user")
	}

	if part != nil && len(part.FileName()) > 0 {
		var msg string
		if asCreate {
			msg = "ObjectMetadata is required during create"
		} else {
			msg = "Metadata must be provided in part named 'ObjectMetadata' to create or update an object and must appear before the file contents"
		}
		if !parsedMetadata {
			return drainFunc, NewAppError(400, nil, msg)
		}
		// Guess the content type and name if it wasn't supplied
		if obj.ContentType.Valid == false || len(obj.ContentType.String) == 0 {
			obj.ContentType.String = guessContentType(part.FileName())
		}
		if obj.Name == "" {
			obj.Name = part.FileName()
		}
		drainFunc, herr, err = h.beginUpload(ctx, caller, part, obj, grant)
		if herr != nil {
			return nil, herr
		}
		if err != nil {
			return nil, NewAppError(500, err, "error caching file")
		}
	}

	// catch the nil,nil,nil return case
	if drainFunc == nil {
		return nil, NewAppError(400, nil, "file must be supplied as multipart mime part")
	}
	return drainFunc, nil
}

func (h AppServer) beginUpload(ctx context.Context, caller Caller, part *multipart.Part, obj *models.ODObject, grant *models.ODObjectPermission) (beginDrain func(), herr *AppError, err error) {
	beganAt := h.Tracker.BeginTime(performance.UploadCounter)
	drainFunc, herr, err := h.beginUploadTimed(ctx, caller, part, obj, grant)
	bytes := obj.ContentSize.Int64
	//If this failed, then don't count it in our statistics
	if herr != nil {
		bytes = 0
	}
	h.Tracker.EndTime(performance.UploadCounter, beganAt, performance.SizeJob(bytes))
	if herr != nil {
		return nil, herr, err
	}
	//Make sure that we can compute throughput from this (the message name and param name are important)
	LoggerFromContext(ctx).Info("transaction up", zap.Int64("bytes", bytes))
	return drainFunc, herr, err
}

func (h AppServer) beginUploadTimed(ctx context.Context, caller Caller, part *multipart.Part, obj *models.ODObject,
	grant *models.ODObjectPermission) (beginDrain func(), herr *AppError, err error) {
	logger := LoggerFromContext(ctx)
	dp := ciphertext.FindCiphertextCacheByObject(obj)
	masterKey := dp.GetMasterKey()
	fileID := ciphertext.FileId(obj.ContentConnector.String)
	iv := obj.EncryptIV
	// TODO this is where we actually use grant.
	fileKey := crypto.ApplyPassphrase(masterKey, grant.PermissionIV, grant.EncryptKey)
	d := ciphertext.FindCiphertextCacheByObject(obj)

	// CiphertextCacheFilesystemMountPoint.Resolve(FileName) returns a path.
	outFileUploading := d.Resolve(ciphertext.NewFileName(fileID, ".uploading"))
	outFileUploaded := d.Resolve(ciphertext.NewFileName(fileID, ".uploaded"))

	outFile, err := d.Files().Create(outFileUploading)
	if err != nil {
		msg := fmt.Sprintf("Unable to open ciphertext uploading file %s", outFileUploading)
		return nil, NewAppError(500, err, msg), err
	}
	defer outFile.Close()

	//Write the encrypted data to the filesystem
	byteRange := crypto.NewByteRange()
	checksum, length, err := crypto.DoCipherByReaderWriter(logger, part, outFile, fileKey, iv, "uploading from browser", byteRange)
	if err != nil {
		//It could be the client's fault, so we use 400 here.
		msg := fmt.Sprintf("Unable to write ciphertext %s", outFileUploading)
		//If something went wrong, just get rid of this file.  We only have part of it,
		// so we can't retry anyway.
		d.Files().Remove(outFileUploading)
		return nil, NewAppError(400, err, msg), err
	}

	//Rename it to indicate that it can be moved to S3
	err = d.Files().Rename(outFileUploading, outFileUploaded)
	if err != nil {
		msg := fmt.Sprintf("Unable to rename uploaded file %s", outFileUploading)
		// I can't see why this would happen, but this file is toast if this happens.
		d.Files().Remove(outFileUploading)
		return nil, NewAppError(500, err, msg), err
	}
	logger.Info("s3 enqueued", zap.String("fileID", string(fileID)))

	//Record metadata
	obj.ContentHash = checksum
	obj.ContentSize.Int64 = length

	//At the end of this function, we can mark the file as stored in S3.
	return func() { h.Writeback(obj, fileID, length, 3) }, nil, err
}

// Writeback wraps a WritebackAttempt with performance tracking
func (h AppServer) Writeback(obj *models.ODObject, rName ciphertext.FileId, size int64, tries int) error {
	beganAt := h.Tracker.BeginTime(performance.S3DrainTo)
	err := h.WritebackAttempt(obj, rName, size, tries)
	h.Tracker.EndTime(performance.S3DrainTo, beganAt, performance.SizeJob(size))
	return err
}

// WritebackAttempt usues the ciphertextcache associated by an object and writes the content stream to the drain
func (h AppServer) WritebackAttempt(obj *models.ODObject, rName ciphertext.FileId, size int64, tries int) error {
	d := ciphertext.FindCiphertextCacheByObject(obj)
	err := d.Writeback(rName, size)
	tries = tries - 1
	if err != nil {
		//The problem is that we get things like transient DNS errors,
		//after we took custody of the file.  We will need something
		//more robust than this eventually.  We have the file, while
		//not being uploaded if all attempts fail.
		if tries > 0 {
			log.Printf("unable to drain file.  Trying again:%v", err)
			err = h.WritebackAttempt(obj, rName, size, tries)
		} else {
			log.Printf("unable to drain file.  Trying it in the background:%v", err)
			//If we don't want to give up and lose the data, we have to keep trying in another goroutine to avoid blowing up the stack
			go func() {
				//If we are having a drain outage, then trying immediately is not going to be useful.
				//Wait a while
				log.Printf("waiting 30 seconds before attempting to drain again")
				time.Sleep(30 * time.Second)
				log.Printf("drain attempt beginning")
				h.Writeback(obj, rName, size, 3)
			}()
		}
	}
	return err
}

func compareIDFromJSONWithURI(ctx context.Context, obj protocol.UpdateObjectAndStreamRequest) *AppError {

	captured, _ := CaptureGroupsFromContext(ctx)

	fromURI := captured["objectId"]

	if strings.Compare(obj.ID, fromURI) != 0 {
		return NewAppError(400, nil, "ID mismatch: json POST vs. URI")
	}

	return nil
}

func extIs(name string, ext string) bool {
	return strings.ToLower(path.Ext(name)) == strings.ToLower(ext)
}

//GuessContentType will give a best guess if content type not given otherwise
func guessContentType(name string) string {
	contentType := "text/plain"
	switch {
	case extIs(name, ".js"):
		contentType = "application/javascript"
	case extIs(name, ".css"):
		contentType = "text/css"
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
	case extIs(name, ".jpeg"):
		contentType = "image/jpeg"
	case extIs(name, ".png"):
		contentType = "image/png"
	case extIs(name, ".gif"):
		contentType = "image/gif"
	case extIs(name, ".bmp"):
		contentType = "image/bmp"
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
	return contentType
}

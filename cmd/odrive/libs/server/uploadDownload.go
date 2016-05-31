package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path"
	"strings"

	"github.com/uber-go/zap"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/protocol"
)

func (h AppServer) acceptObjectUpload(
	ctx context.Context,
	multipartReader *multipart.Reader,
	obj *models.ODObject,
	grant *models.ODObjectPermission,
	asCreate bool,
) (func(), *AppError, error) {
	var drainFunc func()
	var herr *AppError
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return drainFunc, NewAppError(400, nil, "Could not determine user"), fmt.Errorf("User not provided in context")
	}
	parsedMetadata := false
	var createObjectRequest protocol.Object
	for {
		part, err := multipartReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break //just an eof...not an error
			} else {
				return drainFunc, NewAppError(400, err, "error getting a part"), err
			}
		} // if err != nil

		switch {
		case part.FormName() == "ObjectMetadata":
			//This ID we got off of the URI, because we haven't parsed JSON yet
			existingID := hex.EncodeToString(obj.ID)
			existingParentID := hex.EncodeToString(obj.ParentID)

			s, herr := getFormValueAsString(part)
			if herr != nil {
				return drainFunc, herr, nil
			}

			//It's the same as the database object, but this function might be
			//dealing with a retrieved object, so we get fields individually
			err := json.Unmarshal([]byte(s), &createObjectRequest)
			if err != nil {
				return drainFunc, NewAppError(400, err, fmt.Sprintf("Could not decode ObjectMetadata: %s", s)), err
			}

			// If updating and ACM provided differs from what is currently set, then need to
			// Check AAC to compare user clearance to NEW metadata Classifications
			// to see if allowed for this user
			rawAcmString := createObjectRequest.RawAcm.(string)
			if !asCreate && strings.Compare(obj.RawAcm.String, rawAcmString) != 0 {
				// Ensure user is allowed this acm
				updateObjectRequest := models.ODObject{}
				updateObjectRequest.RawAcm.String = rawAcmString
				updateObjectRequest.RawAcm.Valid = true
				hasAACAccessToNewACM, err := h.isUserAllowedForObjectACM(ctx, &updateObjectRequest)
				if err != nil {
					return drainFunc, NewAppError(500, err, "Error communicating with authorization service"), err
				}
				if !hasAACAccessToNewACM {
					return drainFunc, NewAppError(403, nil, "Unauthorized"), err
				}
			}

			err = mapping.OverwriteODObjectWithProtocolObject(obj, &createObjectRequest)
			if err != nil {
				return drainFunc, NewAppError(400, err, "Could not extract data from json response"), err
			}

			//If this is a new object, check prerequisites
			if asCreate {
				if herr := handleCreatePrerequisites(ctx, h, obj); herr != nil {
					return nil, herr, nil
				}
				if len(obj.RawAcm.String) == 0 {
					return drainFunc, NewAppError(400, err, "An ACM must be specified"), nil
				}
			} else {
				// If the id is specified, it must be the same as from the URI
				if len(createObjectRequest.ID) > 0 && createObjectRequest.ID != existingID {
					return drainFunc, NewAppError(400, err, "JSON supplied an object id inconsistent with the one supplied from URI"), nil
				}
				//Parent id change must not be allowed, as it would let users move the object
				if len(createObjectRequest.ParentID) > 0 && createObjectRequest.ParentID != existingParentID {
					return drainFunc, NewAppError(400, err, "JSON supplied a parent id inconsistent with existing parent id"), nil
				}
			}
			parsedMetadata = true
		case len(part.FileName()) > 0:
			var msg string
			if asCreate {
				msg = "ObjectMetadata is required during create"
			} else {
				msg = "Metadata must be provided in part named 'ObjectMetadata' to create or update an object"
			}
			if !parsedMetadata {
				return drainFunc, NewAppError(400, nil, msg), nil
			}
			if !asCreate {
				if obj.ChangeToken != createObjectRequest.ChangeToken {
					return drainFunc, NewAppError(400, nil, "Changetoken must be up to date"), nil
				}
			}
			//Guess the content type and name if it wasn't supplied
			if obj.ContentType.Valid == false || len(obj.ContentType.String) == 0 {
				obj.ContentType.String = guessContentType(part.FileName())
			}
			if obj.Name == "" {
				obj.Name = part.FileName()
			}
			drainFunc, herr, err = h.beginUpload(ctx, caller, part, obj, grant)
			if herr != nil {
				return nil, herr, err
			}
			if err != nil {
				return drainFunc, NewAppError(500, err, "error caching file"), err
			}
		} // switch
	} //for
	return drainFunc, nil, nil
}

func (h AppServer) beginUpload(
	ctx context.Context,
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	grant *models.ODObjectPermission,
) (beginDrain func(), herr *AppError, err error) {

	beganAt := h.Tracker.BeginTime(performance.UploadCounter)
	drainFunc, herr, err := h.beginUploadTimed(ctx, caller, part, obj, grant)
	h.Tracker.EndTime(performance.UploadCounter, beganAt, performance.SizeJob(obj.ContentSize.Int64))
	if herr != nil {
		return nil, herr, err
	}
	return drainFunc, herr, err
}

func (h AppServer) beginUploadTimed(
	ctx context.Context,
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	grant *models.ODObjectPermission,
) (beginDrain func(), herr *AppError, err error) {
	//
	// Note that since errors here *can* be caused by the client dropping, we will make these 4xx
	// error codes and blame the client for the moment.  When reading from the client and writing
	// to disk, sometimes it is ambiguous who is to blame.  This is similar to the case of a failed
	// lookup when the client may have given us bad information in the lookup.  In these cases,
	// it may be ok to use 400 error codes until proven otherwise.
	//
	rName := FileId(obj.ContentConnector.String)
	iv := obj.EncryptIV
	fileKey := grant.EncryptKey
	d := h.DrainProvider

	//Make up a random name for our file - don't deal with versioning yet
	outFileUploading := d.Resolve(NewFileName(rName, ".uploading"))
	outFileUploaded := d.Resolve(NewFileName(rName, ".uploaded"))

	outFile, err := d.Files().Create(outFileUploading)
	if err != nil {
		msg := fmt.Sprintf("Unable to open ciphertext uploading file %s", outFileUploading)
		return nil, NewAppError(500, err, msg), err
	}
	defer outFile.Close()

	//Write the encrypted data to the filesystem
	byteRange := utils.NewByteRange()
	checksum, length, err := utils.DoCipherByReaderWriter(part, outFile, fileKey, iv, "uploading from browser", byteRange)
	if err != nil {
		//It could be the client's fault, so we use 400 here.
		msg := fmt.Sprintf("Unable to write ciphertext %s", outFileUploading)
		//If something went wrong, just get rid of this file.  We only have part of it,
		// so we can't retry anyway.
		d.Files().Remove(outFileUploading)
		return nil, NewAppError(400, err, msg), err
	}

	//Scramble the fileKey with the masterkey - will need it once more on retrieve
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, fileKey)

	//Rename it to indicate that it can be moved to S3
	err = d.Files().Rename(outFileUploading, outFileUploaded)
	if err != nil {
		msg := fmt.Sprintf("Unable to rename uploaded file %s", outFileUploading)
		// I can't see why this would happen, but this file is toast if this happens.
		d.Files().Remove(outFileUploading)
		return nil, NewAppError(500, err, msg), err
	}
	LoggerFromContext(ctx).Info("rename", zap.String("from", string(outFileUploading)), zap.String("to", string(outFileUploaded)))

	//Record metadata
	obj.ContentHash = checksum
	obj.ContentSize.Int64 = length

	//At the end of this function, we can mark the file as stored in S3.
	return func() { h.cacheToDrain(&config.DefaultBucket, rName, length, 3) }, nil, err
}

//We get penalized on throughput if these fail a lot.
//I think that's reasonable to be measuring "goodput" this way.
func (h AppServer) cacheToDrain(
	bucket *string,
	rName FileId,
	size int64,
	tries int,
) error {
	beganAt := h.Tracker.BeginTime(performance.S3DrainTo)
	err := h.cacheToDrainAttempt(bucket, rName, size, tries)
	h.Tracker.EndTime(performance.S3DrainTo, beganAt, performance.SizeJob(size))
	return err
}

func (h AppServer) cacheToDrainAttempt(
	bucket *string,
	rName FileId,
	size int64,
	tries int,
) error {
	d := h.DrainProvider
	err := d.CacheToDrain(bucket, rName, size)
	tries = tries - 1
	if err != nil {
		//The problem is that we get things like transient DNS errors,
		//after we took custody of the file.  We will need something
		//more robust than this eventually.  We have the file, while
		//not being uploaded if all attempts fail.
		if tries > 0 {
			log.Printf("unable to drain file.  Trying again:%v", err)
			err = h.cacheToDrainAttempt(bucket, rName, size, tries)
		} else {
			log.Printf("unable to drain file.  Giving up and deleting it:%v", err)
			//If we give up, we must delete the file
			uploadedFile := d.Resolve(NewFileName(rName, ".uploaded"))
			d.Files().Remove(uploadedFile)
		}
	}
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
	return contentType
}

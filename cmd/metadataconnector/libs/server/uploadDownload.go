package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/cmd/metadataconnector/libs/utils"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/metadata/models/acm"
	"decipher.com/oduploader/performance"
	"decipher.com/oduploader/protocol"
)

func (h AppServer) acceptObjectUpload(
	ctx context.Context,
	multipartReader *multipart.Reader,
	obj *models.ODObject,
	grant *models.ODObjectPermission,
	asCreate bool,
) (*AppError, error) {
	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return &AppError{Code: 400, Err: nil, Msg: "Could not determine user"}, fmt.Errorf("User not provided in context")
	}
	parsedMetadata := false
	var createObjectRequest protocol.Object
	for {
		part, err := multipartReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break //just an eof...not an error
			} else {
				return &AppError{Code: 400, Err: err, Msg: "error getting a part"}, err
			}
		} // if err != nil

		switch {
		case part.FormName() == "ObjectMetadata":
			//This ID we got off of the URI, because we haven't parsed JSON yet
			existingID := hex.EncodeToString(obj.ID)
			existingParentID := hex.EncodeToString(obj.ParentID)

			s := getFormValueAsString(part)
			//It's the same as the database object, but this function might be
			//dealing with a retrieved object, so we get fields individually
			err := json.Unmarshal([]byte(s), &createObjectRequest)
			if err != nil {
				return &AppError{400, err, fmt.Sprintf("Could not decode ObjectMetadata: %s", s)}, err
			}

			// If updating and ACM provided differs from what is currently set, then need to
			// Check AAC to compare user clearance to NEW metadata Classifications
			// to see if allowed for this user
			if !asCreate && strings.Compare(obj.RawAcm.String, createObjectRequest.RawAcm) != 0 {
				// Validate ACM
				rawAcmString := createObjectRequest.RawAcm
				// Make sure its parseable
				parsedACM, err := acm.NewACMFromRawACM(rawAcmString)
				if err != nil {
					return &AppError{428, nil, "ACM provided could not be parsed"}, err
				}
				// Ensure user is allowed this acm
				updateObjectRequest := models.ODObject{}
				updateObjectRequest.RawAcm.String = createObjectRequest.RawAcm
				hasAACAccessToNewACM, err := h.isUserAllowedForObjectACM(ctx, &updateObjectRequest)
				if err != nil {
					return &AppError{500, nil, "Error communicating with authorization service"}, err
				}
				if !hasAACAccessToNewACM {
					return &AppError{403, nil, "Unauthorized"}, err
				}
				// Capture values before the mapping
				acmID := obj.ACM.ID
				acmACMID := obj.ACM.ACMID
				acmObjectID := obj.ACM.ObjectID
				// Map the parsed acm
				obj.ACM = mapping.MapACMToODObjectACM(&parsedACM)
				// Assign existinng values back over top
				obj.ACM.ID = acmID
				obj.ACM.ACMID = acmACMID
				obj.ACM.ObjectID = acmObjectID
				obj.ACM.ModifiedBy = caller.DistinguishedName
			}

			err = mapping.OverwriteODObjectWithProtocolObject(obj, &createObjectRequest)
			if err != nil {
				return &AppError{400, err, "Could not extract data from json response"}, err
			}

			//If this is a new object, check prerequisites
			if asCreate {
				if herr := handleCreatePrerequisites(ctx, h, obj); herr != nil {
					return herr, nil
				}
				if len(obj.RawAcm.String) == 0 {
					return &AppError{400, err, "An ACM must be specified"}, nil
				}
			} else {
				// If the id is specified, it must be the same as from the URI
				if len(createObjectRequest.ID) > 0 && createObjectRequest.ID != existingID {
					return &AppError{
						Code: 400,
						Err:  err,
						Msg:  "JSON supplied an object id inconsistent with the one supplied from URI",
					}, nil
				}
				//Parent id change must not be allowed, as it would let users move the object
				if len(createObjectRequest.ParentID) > 0 && createObjectRequest.ParentID != existingParentID {
					return &AppError{
						Code: 400,
						Err:  err,
						Msg:  "JSON supplied an parent id",
					}, nil
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
				return &AppError{
					Code: 400,
					Err:  nil,
					Msg:  msg,
				}, nil
			}
			if !asCreate {
				if obj.ChangeToken != createObjectRequest.ChangeToken {
					return &AppError{
						Code: 400,
						Err:  nil,
						Msg:  "Changetoken must be up to date",
					}, nil
				}
			}
			//Guess the content type and name if it wasn't supplied
			if obj.ContentType.Valid == false || len(obj.ContentType.String) == 0 {
				obj.ContentType.String = guessContentType(part.FileName())
			}
			if obj.Name == "" {
				obj.Name = part.FileName()
			}
			async := true
			herr, err := h.beginUpload(caller, part, obj, grant, async)
			if herr != nil {
				return herr, err
			}
			if err != nil {
				return &AppError{500, err, "error caching file"}, err
			}
		} // switch
	} //for
	return nil, nil
}

func (h AppServer) beginUpload(
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	grant *models.ODObjectPermission,
	async bool,
) (herr *AppError, err error) {

	beganAt := h.Tracker.BeginTime(performance.UploadCounter)
	herr, err = h.beginUploadTimed(caller, part, obj, grant, async)
	if herr != nil {
		h.Tracker.EndTime(
			performance.UploadCounter,
			beganAt,
			performance.SizeJob(obj.ContentSize.Int64),
		)
		return herr, err
	}
	h.Tracker.EndTime(
		performance.UploadCounter,
		beganAt,
		performance.SizeJob(obj.ContentSize.Int64),
	)
	return herr, err
}

func (h AppServer) beginUploadTimed(
	caller Caller,
	part *multipart.Part,
	obj *models.ODObject,
	grant *models.ODObjectPermission,
	async bool,
) (herr *AppError, err error) {
	rName := obj.ContentConnector.String
	iv := obj.EncryptIV
	fileKey := grant.EncryptKey

	//Make up a random name for our file - don't deal with versioning yet
	outFileUploading := h.DrainProvider.CacheLocation() + "/" + rName + ".uploading"
	outFileUploaded := h.DrainProvider.CacheLocation() + "/" + rName + ".uploaded"

	outFile, err := os.Create(outFileUploading)
	if err != nil {
		log.Printf("Unable to open ciphertext uploading file %s %v:", outFileUploading, err)
		return nil, err
	}
	defer outFile.Close()

	//Write the encrypted data to the filesystem
	checksum, length, err := utils.DoCipherByReaderWriter(part, outFile, fileKey, iv, "uploading from browser")
	if err != nil {
		log.Printf("Unable to write ciphertext %s %v:", outFileUploading, err)
		return nil, err
	}

	//Scramble the fileKey with the masterkey - will need it once more on retrieve
	utils.ApplyPassphrase(h.MasterKey+caller.DistinguishedName, fileKey)

	//Rename it to indicate that it can be moved to S3
	err = os.Rename(outFileUploading, outFileUploaded)
	if err != nil {
		log.Printf("Unable to rename uploaded file %s %v:", outFileUploading, err)
		return nil, err
	}
	log.Printf("rename:%s -> %s", outFileUploading, outFileUploaded)

	//Record metadata
	log.Printf("checksum:%s", hex.EncodeToString(checksum))
	obj.ContentHash = checksum
	obj.ContentSize.Int64 = length

	//Just return 200 when we run async, because the client tells
	//us whether it's async or not.
	if async {
		go h.cacheToDrain(&config.DefaultBucket, rName, length, 3)
	} else {
		h.cacheToDrain(&config.DefaultBucket, rName, length, 3)
	}
	return nil, err
}

//We get penalized on throughput if these fail a lot.
//I think that's reasonable to be measuring "goodput" this way.
func (h AppServer) cacheToDrain(
	bucket *string,
	rName string,
	size int64,
	tries int,
) error {
	beganAt := h.Tracker.BeginTime(performance.S3DrainTo)
	err := h.cacheToDrainAttempt(bucket, rName, size, tries)
	h.Tracker.EndTime(
		performance.S3DrainTo,
		beganAt,
		performance.SizeJob(size),
	)
	return err
}

func (h AppServer) cacheToDrainAttempt(
	bucket *string,
	rName string,
	size int64,
	tries int,
) error {
	err := h.DrainProvider.CacheToDrain(bucket, rName, size)
	tries = tries - 1
	if err != nil {
		//The problem is that we get things like transient DNS errors,
		//after we took custody of the file.  We will need something
		//more robust than this eventually.  We have the file, while
		//not being uploaded if all attempts fail.
		log.Printf("unable to drain file.  Trying again:%v", err)
		if tries > 0 {
			err = h.cacheToDrainAttempt(bucket, rName, size, tries)
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
	log.Printf("uploaded %s as %s", name, contentType)
	return contentType
}

func (h AppServer) drainToCache(
	bucket *string,
	theFile string,
	length int64,
) (*AppError, error) {
	beganAt := h.Tracker.BeginTime(performance.S3DrainFrom)
	herr, err := h.DrainProvider.DrainToCache(bucket, theFile)
	if herr != nil {
		h.Tracker.EndTime(
			performance.S3DrainFrom,
			beganAt,
			performance.SizeJob(length),
		)
		return herr, err
	}
	return nil, nil
}

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

	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/crypto"
	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/mapping"

	"mime"

	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/performance"
	"github.com/deciphernow/object-drive-server/protocol"
)

// If we are returning potentially after the object has been uploaded to disk,
// then there is a time-span where abort must cleanup after itself
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

func (h AppServer) acceptObjectUpload(ctx context.Context, mpr *multipart.Reader, obj *models.ODObject,
	grant *models.ODObjectPermission, asCreate bool, afterMeta func(*models.ODObject)) (func(), string, bool, *AppError) {

	part, err := mpr.NextPart()
	if err != nil {
		return nil, "", false, NewAppError(400, err, "error getting metadata part")
	}

	parsedMetadata, pathDelimiter, recursive, herr := h.acceptObjectUploadMeta(ctx, part, obj, grant, asCreate)
	if herr != nil {
		return nil, "", false, herr
	}

	// Get the second part if the first was consumed.
	if parsedMetadata {
		part, err = mpr.NextPart()
		if err == io.EOF {
			return nil, "", false, NewAppError(400, err, "error getting stream part")
		}
	}

	//This is code inserted in between metadata parse and accepting the stream
	if afterMeta != nil {
		afterMeta(obj)
	}

	// Process the stream
	drainFunc, herr := h.acceptObjectUploadStream(ctx, part, obj, grant)
	return drainFunc, pathDelimiter, recursive, herr
}

// Get an update obj from the caller - we are not persisting anything yet
func (h AppServer) acceptObjectUploadMeta(ctx context.Context, part *multipart.Part, obj *models.ODObject,
	grant *models.ODObjectPermission, asCreate bool) (bool, string, bool, *AppError) {
	var herr *AppError

	var recursive bool

	parsedMetadata := false
	var createObjectRequest protocol.CreateObjectRequest
	var updateObjectRequest protocol.UpdateObjectAndStreamRequest
	var pathDelimiter string

	if part.FormName() == "ObjectMetadata" {
		parsedMetadata = true

		limit := 5 << (10 * 2)
		metadata, err := ioutil.ReadAll(io.LimitReader(part, int64(limit)))
		if err != nil {
			return parsedMetadata, "", false, NewAppError(400, err, "could not read json metadata")
		}
		// Parse into a useable struct
		if asCreate {
			if err = json.Unmarshal(metadata, &createObjectRequest); err != nil {
				return parsedMetadata, "", false, NewAppError(400, err, fmt.Sprintf("Could not decode ObjectMetadata: %s", metadata))
			}
			// Mapping to object
			err = mapping.OverwriteODObjectWithCreateObjectRequest(obj, &createObjectRequest)
			if err != nil {
				return parsedMetadata, "", false, NewAppError(400, err, fmt.Sprintf("Error creating object with data from request: %s", err.Error()))
			}
			// Post mapping rules applied for create (not deleted, enforce owner cruds, assign meta)
			if herr := handleCreatePrerequisites(ctx, h, obj); herr != nil {
				return parsedMetadata, "", false, herr
			}
			pathDelimiter = createObjectRequest.NamePathDelimiter
		} else {
			if err = json.Unmarshal(metadata, &updateObjectRequest); err != nil {
				return parsedMetadata, "", false, NewAppError(400, err, fmt.Sprintf("Could not decode ObjectMetadata: %s", metadata))
			}
			// ID in json must match that on the URI
			herr = compareIDFromJSONWithURI(ctx, updateObjectRequest)
			if herr != nil {
				return parsedMetadata, "", false, herr
			}
			// ChangeToken must be provided and match the object
			if obj.ChangeToken != updateObjectRequest.ChangeToken {
				return parsedMetadata, "", false, NewAppError(400, nil, "Changetoken must be up to date")
			}
			// Mapping to object
			err = mapping.OverwriteODObjectWithUpdateObjectAndStreamRequest(obj, &updateObjectRequest)
			if err != nil {
				return parsedMetadata, "", false, NewAppError(400, err, fmt.Sprintf("Could not extract data from json response: %s", err.Error()))
			}
			// Set our recursive bool here
			recursive = updateObjectRequest.RecursiveShare
		}

		// Whether creating or updating, the ACM must have a value
		if len(obj.RawAcm.String) == 0 {
			return parsedMetadata, "", false, NewAppError(400, err, "An ACM must be specified")
		}
	}
	return parsedMetadata, pathDelimiter, recursive, herr
}

func (h AppServer) parsePartContentType(part *multipart.Part) *AppError {
	cte := part.Header.Get("Content-Transfer-Encoding")
	if len(cte) > 0 {
		cte = strings.ToLower(cte)
		if cte != "binary" && cte != "8bit" && cte != "7bit" {
			msg := fmt.Sprintf("Content-Transfer-Encoding: %s is not supported for file part. File should be provided in native binary format.", cte)
			return NewAppError(400, fmt.Errorf("%s", msg), msg)
		}
	}
	ct := part.Header.Get("Content-Type")
	ctparts := strings.Split(ct, ";")
	if len(ctparts) > 1 {
		for _, v := range ctparts {
			lv := strings.ToLower(strings.TrimSpace(v))
			// Sampling from `Content-Type: image/png; base64`. Not even sure this is valid in this header, as its more typical in html img src.
			if lv == "base64" {
				msg := fmt.Sprintf("Content-Type: %s is not supported for file part. File should be provided in native binary format with no encoding declarations.", ct)
				return NewAppError(400, fmt.Errorf("%s", msg), msg)
			}
			// Permit character set, but only if utf-8 or charset=ISO-8859-1
			if strings.HasPrefix(lv, "charset=") {
				if lv != "charset=utf-8" && lv != "charset=iso-8859-1" {
					msg := fmt.Sprintf("Content-Type: %s is not supported for file part. File should be provided in native binary format with no encoding declarations.", ct)
					return NewAppError(400, fmt.Errorf("%s", msg), msg)
				}
			}
		}
	}
	return nil
}

// Get the bytes from the caller.
func (h AppServer) acceptObjectUploadStream(ctx context.Context, part *multipart.Part, obj *models.ODObject,
	grant *models.ODObjectPermission) (func(), *AppError) {

	var herr *AppError
	var err error
	var drainFunc func()

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return nil, NewAppError(400, fmt.Errorf("User not provided in context"), "Could not determine user")
	}

	if part == nil || len(part.FileName()) == 0 {
		if drainFunc == nil {
			return nil, NewAppError(400, nil, "file must be supplied as multipart mime part")
		}
		return drainFunc, nil
	}

	// Guess the content type and name if it wasn't supplied
	if obj.ContentType.Valid == false || len(obj.ContentType.String) == 0 {
		obj.ContentType = models.ToNullString(GetContentTypeFromFilename(part.FileName()))
	}
	if obj.Name == "" {
		obj.Name = part.FileName()
	}
	// Issue #663, #739 Look to see if any encoding is set and isn't binary
	herr = h.parsePartContentType(part)
	if herr != nil {
		return nil, herr
	}

	drainFunc, herr, err = h.beginUpload(ctx, caller, part, obj, grant)
	if herr != nil {
		return nil, herr
	}
	if err != nil {
		return nil, NewAppError(500, err, "error caching file")
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

	// Write the encrypted data to the filesystem
	byteRange := crypto.NewByteRange()
	checksum, length, err := crypto.DoCipherByReaderWriter(logger, part, outFile, fileKey, iv, "uploading from browser", byteRange)
	if err != nil {
		// It could be the client's fault.  Check the message.
		msg := fmt.Sprintf("Unable to write ciphertext %s", outFileUploading)
		// If something went wrong, just get rid of this file.  We only have part of it,
		// so we can't retry anyway.
		d.Files().Remove(outFileUploading)
		// The user terminating the upload is actually not an internal error, and user can trigger it intentionally
		if strings.HasPrefix(err.Error(), "multipart: Part Read: unexpected EOF") {
			return nil, NewAppError(400, err, msg), err
		}
		return nil, NewAppError(500, err, msg), err
	}

	// Rename it to indicate that it can be moved to S3
	err = d.Files().Rename(outFileUploading, outFileUploaded)
	if err != nil {
		msg := fmt.Sprintf("Unable to rename uploaded file %s", outFileUploading)
		// I can't see why this would happen, but this file is toast if this happens.
		d.Files().Remove(outFileUploading)
		return nil, NewAppError(500, err, msg), err
	}
	logger.Info("s3 enqueued", zap.String("fileID", string(fileID)))

	// Record metadata
	obj.ContentHash = checksum
	obj.ContentSize.Int64 = length

	// At the end of this function, we can mark the file as stored in S3.
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

// ExtensionToContentType provides a mapping of content type from a provided file extension
var ExtensionToContentType map[string]string

func populateExtensionToContentTypeMap() {
	ExtensionToContentType = make(map[string]string)
	ExtensionToContentType["3gp"] = "video/3gpp"
	ExtensionToContentType["7z"] = "application/x-7z-compressed"
	ExtensionToContentType["ai"] = "application/postscript"
	ExtensionToContentType["anx"] = "application/annodex"
	ExtensionToContentType["atom"] = "application/atom+xml"
	ExtensionToContentType["avi"] = "video/x-msvideo"
	ExtensionToContentType["axa"] = "audio/annodex"
	ExtensionToContentType["axv"] = "video/annodex"
	ExtensionToContentType["bat"] = "application/x-msdos-program"
	ExtensionToContentType["bmp"] = "image/bmp"
	ExtensionToContentType["c++"] = "text/x-c++src"
	ExtensionToContentType["cab"] = "application/x-cab"
	ExtensionToContentType["cc"] = "text/x-c++src"
	ExtensionToContentType["cdr"] = "image/x-coreldraw"
	ExtensionToContentType["class"] = "application/java-vm"
	ExtensionToContentType["com"] = "application/x-msdos-program"
	ExtensionToContentType["cpp"] = "text/x-c++src"
	ExtensionToContentType["css"] = "text/css"
	ExtensionToContentType["csv"] = "text/csv"
	ExtensionToContentType["cxx"] = "text/x-c++src"
	ExtensionToContentType["dll"] = "application/x-msdos-program"
	ExtensionToContentType["doc"] = "application/msword"
	ExtensionToContentType["docm"] = "application/vnd.ms-word.document.macroEnabled.12"
	ExtensionToContentType["docx"] = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	ExtensionToContentType["dot"] = "application/msword"
	ExtensionToContentType["dotm"] = "application/vnd.ms-word.template.macroEnabled.12"
	ExtensionToContentType["dotx"] = "application/vnd.openxmlformats-officedocument.wordprocessingml.template"
	ExtensionToContentType["eml"] = "message/rfc822"
	ExtensionToContentType["eps"] = "application/postscript"
	ExtensionToContentType["eps2"] = "application/postscript"
	ExtensionToContentType["eps3"] = "application/postscript"
	ExtensionToContentType["epsf"] = "application/postscript"
	ExtensionToContentType["epsi"] = "application/postscript"
	ExtensionToContentType["exe"] = "application/x-msdos-program"
	ExtensionToContentType["flac"] = "audio/flac"
	ExtensionToContentType["flv"] = "video/x-flv"
	ExtensionToContentType["gif"] = "image/gif"
	ExtensionToContentType["gz"] = "application/gzip"
	ExtensionToContentType["hta"] = "application/hta"
	ExtensionToContentType["htm"] = "text/html"
	ExtensionToContentType["html"] = "text/html"
	ExtensionToContentType["hs"] = "text/x-haskell"
	ExtensionToContentType["ico"] = "image/x-icon"
	ExtensionToContentType["ief"] = "image/ief"
	ExtensionToContentType["jar"] = "application/java-archive"
	ExtensionToContentType["java"] = "text/x-java"
	ExtensionToContentType["jp2"] = "image/jp2"
	ExtensionToContentType["jpe"] = "image/jpeg"
	ExtensionToContentType["jpeg"] = "image/jpeg"
	ExtensionToContentType["jpg"] = "image/jpeg"
	ExtensionToContentType["jpg2"] = "image/jp2"
	ExtensionToContentType["js"] = "application/javascript"
	ExtensionToContentType["json"] = "application/json"
	ExtensionToContentType["kar"] = "audio/midi"
	ExtensionToContentType["kml"] = "application/vnd.google-earth.kml+xml"
	ExtensionToContentType["kmz"] = "application/vnd.google-earth.kmz"
	ExtensionToContentType["m4a"] = "audio/mpeg"
	ExtensionToContentType["m4v"] = "video/mp4"
	ExtensionToContentType["md"] = "text/markdown"
	ExtensionToContentType["mdb"] = "application/msaccess"
	ExtensionToContentType["mid"] = "audio/midi"
	ExtensionToContentType["midi"] = "audio/midi"
	ExtensionToContentType["mov"] = "video/mov"
	ExtensionToContentType["mp2"] = "audio/mpeg"
	ExtensionToContentType["mp3"] = "audio/mp3"
	ExtensionToContentType["mp4"] = "video/mp4"
	ExtensionToContentType["mpe"] = "video/mpeg"
	ExtensionToContentType["mpeg"] = "video/mpeg"
	ExtensionToContentType["mpega"] = "audio/mpeg"
	ExtensionToContentType["mpg"] = "video/mpeg"
	ExtensionToContentType["mpga"] = "audio/mpeg"
	ExtensionToContentType["mpv"] = "video/mpv"
	ExtensionToContentType["odb"] = "application/vnd.oasis.opendocument.database"
	ExtensionToContentType["odc"] = "application/vnd.oasis.opendocument.chart"
	ExtensionToContentType["odf"] = "application/vnd.oasis.opendocument.formula"
	ExtensionToContentType["odg"] = "application/vnd.oasis.opendocument.graphics"
	ExtensionToContentType["odi"] = "application/vnd.oasis.opendocument.image"
	ExtensionToContentType["odp"] = "application/vnd.oasis.opendocument.presentation"
	ExtensionToContentType["odm"] = "application/vnd.oasis.opendocument.text-master"
	ExtensionToContentType["ods"] = "application/vnd.oasis.opendocument.shreadsheet"
	ExtensionToContentType["odt"] = "application/vnd.oasis.opendocument.text"
	ExtensionToContentType["oga"] = "audio/ogg"
	ExtensionToContentType["ogg"] = "video/ogg" // WARNING: may also be "audio/ogg"
	ExtensionToContentType["ogv"] = "video/ogg"
	ExtensionToContentType["ogx"] = "application/ogg"
	ExtensionToContentType["opus"] = "audio/ogg"
	ExtensionToContentType["otg"] = "application/vnd.oasis.opendocument.graphics-template"
	ExtensionToContentType["oth"] = "application/vnd.oasis.opendocument.text-web"
	ExtensionToContentType["otp"] = "application/vnd.oasis.opendocument.presentation-template"
	ExtensionToContentType["ots"] = "application/vnd.oasis.opendocument.shreadsheet-template"
	ExtensionToContentType["ott"] = "application/vnd.oasis.opendocument.text-template"
	ExtensionToContentType["pbm"] = "image/x-portable-bitmap"
	ExtensionToContentType["pcx"] = "image/pcx"
	ExtensionToContentType["pdf"] = "application/pdf"
	ExtensionToContentType["png"] = "image/png"
	ExtensionToContentType["postscript"] = "application/postscript"
	ExtensionToContentType["potm"] = "application/vnd.ms-powerpoint.template.macroEnabled.12"
	ExtensionToContentType["potx"] = "application/vnd.openxmlformats-officedocument.presentationml.template"
	ExtensionToContentType["ppam"] = "application/vnd.ms-powerpoint.addin.macroEnabled.12"
	ExtensionToContentType["ppsm"] = "application/vnd.ms-powerpoint.slideshow.macroEnabled.12"
	ExtensionToContentType["pps"] = "application/vnd.ms-powerpoint"
	ExtensionToContentType["ppsx"] = "application/vnd.openxmlformats-officedocument.presentationml.slideshow"
	ExtensionToContentType["ppt"] = "application/vnd.ms-powerpoint"
	ExtensionToContentType["pptm"] = "application/vnd.ms-powerpoint.presentation.macroEnabled.12"
	ExtensionToContentType["pptx"] = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	ExtensionToContentType["ps"] = "application/postscript"
	ExtensionToContentType["psd"] = "image/x-photoshop"
	ExtensionToContentType["py"] = "text/x-python"
	ExtensionToContentType["rar"] = "application/x-rar-compressed"
	ExtensionToContentType["rss"] = "application/x-rss+xml"
	ExtensionToContentType["rtf"] = "text/rtf"
	ExtensionToContentType["scala"] = "text/x-scala"
	ExtensionToContentType["sda"] = "application/vnd.stardivision.draw"
	ExtensionToContentType["sdc"] = "application/vnd.stardivision.calc"
	ExtensionToContentType["sdd"] = "application/vnd.stardivision.impress"
	ExtensionToContentType["sdf"] = "application/vnd.stardivision.math"
	ExtensionToContentType["sds"] = "application/vnd.stardivision.chart"
	ExtensionToContentType["sdw"] = "application/vnd.stardivision.writer"
	ExtensionToContentType["sgl"] = "application/vnd.stardivision.writer-global"
	ExtensionToContentType["sgml"] = "text/sgml"
	ExtensionToContentType["sldm"] = "application/vnd.ms-powerpoint.slide.macroEnabled.12"
	ExtensionToContentType["sldx"] = "application/vnd.openxmlformats-officedocument.presentationml.slide"
	ExtensionToContentType["spx"] = "audio/ogg"
	ExtensionToContentType["sql"] = "application/x-sql"
	ExtensionToContentType["stc"] = "application/vnd.sun.xml.calc.template"
	ExtensionToContentType["std"] = "application/vnd.sun.xml.draw.template"
	ExtensionToContentType["sti"] = "application/vnd.sun.xml.impress.template"
	ExtensionToContentType["stw"] = "application/vnd.sun.xml.writer.template"
	ExtensionToContentType["svg"] = "image/svg+xml"
	ExtensionToContentType["svgz"] = "image/svg+xml"
	ExtensionToContentType["sxc"] = "application/vnd.sun.xml.calc"
	ExtensionToContentType["sxd"] = "application/vnd.sun.xml.draw"
	ExtensionToContentType["sxg"] = "application/vnd.sun.xml.writer.global"
	ExtensionToContentType["sxi"] = "application/vnd.sun.xml.impress"
	ExtensionToContentType["sxm"] = "application/vnd.sun.xml.math"
	ExtensionToContentType["sxw"] = "application/vnd.sun.xml.writer"
	ExtensionToContentType["swf"] = "application/x-shockwave-flash"
	ExtensionToContentType["tar"] = "application/x-tar"
	ExtensionToContentType["tif"] = "image/tiff"
	ExtensionToContentType["tiff"] = "image/tiff"
	ExtensionToContentType["torrent"] = "application/x-bittorrent"
	ExtensionToContentType["ttf"] = "application/x-font-ttf"
	ExtensionToContentType["txt"] = "text/plain"
	ExtensionToContentType["vmrl"] = "model/vrml"
	ExtensionToContentType["vsd"] = "application/vnd.visio"
	ExtensionToContentType["vss"] = "application/vnd.visio"
	ExtensionToContentType["vst"] = "application/vnd.visio"
	ExtensionToContentType["vsw"] = "application/vnd.visio"
	ExtensionToContentType["wav"] = "audio/wav"
	ExtensionToContentType["wbmp"] = "image/vnd.wap.wbmp"
	ExtensionToContentType["wml"] = "text/vnd.wap.wml"
	ExtensionToContentType["wmv"] = "video/x-ms-wmv"
	ExtensionToContentType["wmx"] = "video/x-ms-wmx"
	ExtensionToContentType["wp5"] = "application/vnd.wordperfect5.1"
	ExtensionToContentType["wpd"] = "application/vnd.wordperfect"
	ExtensionToContentType["wrl"] = "model/vrml"
	ExtensionToContentType["wvx"] = "video/x-ms-wvx"
	ExtensionToContentType["xlam"] = "application/vnd.ms-excel.addin.macroEnabled.12"
	ExtensionToContentType["xlb"] = "application/vnd.ms-excel"
	ExtensionToContentType["xls"] = "application/vnd.ms-excel"
	ExtensionToContentType["xlsb"] = "application/vnd.ms-excel.sheet.binary.macroEnabled.12"
	ExtensionToContentType["xlsm"] = "application/vnd.ms-excel.sheet.macroEnabled.12"
	ExtensionToContentType["xlsx"] = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	ExtensionToContentType["xlt"] = "application/vnd.ms-excel"
	ExtensionToContentType["xltm"] = "application/vnd.ms-excel.template.macroEnabled.12"
	ExtensionToContentType["xltx"] = "application/vnd.openxmlformats-officedocument.spreadsheetml.template"
	ExtensionToContentType["xml"] = "application/xml"
	ExtensionToContentType["xsd"] = "application/xml"
	ExtensionToContentType["xsl"] = "application/xslt+xml"
	ExtensionToContentType["xslt"] = "application/xslt+xml"
	ExtensionToContentType["xspf"] = "application/xspf+xml"
	ExtensionToContentType["zip"] = "application/zip"
}
func init() {
	populateExtensionToContentTypeMap()
}

// GetContentTypeFromFilename will give a best guess if content type not given otherwise
func GetContentTypeFromFilename(name string) string {

	defaultType := "application/octet-stream"
	extension := strings.ToLower(path.Ext(name))
	if extension == "" {
		return defaultType
	}
	if strings.HasPrefix(extension, ".") {
		extension = extension[1:]
	}
	contentType := ExtensionToContentType[extension]
	// If we didn't get a mapped type, try from system config
	if contentType == "" {
		contentType = mime.TypeByExtension(extension)
	}
	// If still dont have, then use the default
	if contentType == "" {
		contentType = defaultType
	}
	return contentType
}

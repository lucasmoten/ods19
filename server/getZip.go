package server

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"go.uber.org/zap"

	"encoding/hex"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/ciphertext"
	"bitbucket.di2e.net/dime/object-drive-server/crypto"
	"bitbucket.di2e.net/dime/object-drive-server/protocol"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/services/aac"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"golang.org/x/net/context"
)

// zipFileInfo implements a required interface to set time and permissions on files
type zipFileInfo struct {
	size    int64
	name    string
	modTime time.Time
	mode    os.FileMode
	isDir   bool
	sys     interface{}
}

// Implment FileInfo interface for zipFileInfo.
func (z *zipFileInfo) Size() int64        { return z.size }
func (z *zipFileInfo) Name() string       { return z.name }
func (z *zipFileInfo) ModTime() time.Time { return z.modTime }
func (z *zipFileInfo) IsDir() bool        { return z.isDir }
func (z *zipFileInfo) Sys() interface{}   { return z.sys }
func (z *zipFileInfo) Mode() os.FileMode  { return z.mode }

// newFileAcmInfo is everything that a zip needs to know about the file
func newFileAcmInfo(fqPath string, rawAcm string, dt time.Time) *zipFileInfo {
	return &zipFileInfo{
		size:    int64(len(rawAcm)),
		name:    path.Clean(fqPath),
		modTime: dt,
		mode:    os.FileMode(0600),
		isDir:   false,
	}
}

// This is an item in the manifest
type zipManifestItem struct {
	Portion string
	Name    string
}

// All the state we need to write a manifest
// Eventually, we will need rollup info.
type zipManifest struct {
	Files []zipManifestItem
	ACMs  map[string]bool
}

// All the state we need to deconflict names in a directory
type zipUsedNames struct {
	UsedNames map[string]bool
}

// We need to retain classifications of files, so we need to write something
// to re-associate them, short of doing file renaming to retain classification
func zipWriteManifest(ctx context.Context, aacClient aac.AacService, zw *zip.Writer, manifest *zipManifest) *AppError {
	user, ok := UserFromContext(ctx)
	if !ok {
		return NewAppError(http.StatusInternalServerError, nil, "unable to get user from context")
	}
	// manifest acms were stored in a list to make them unique
	// So now we need the list of unique values
	var acmList []string
	for a := range manifest.ACMs {
		acmList = append(acmList, a)
	}

	if len(acmList) == 0 {
		return NewAppError(http.StatusBadRequest, nil, "No data to zip")
	}

	// Compute the rollup
	resp, err := aacClient.RollupAcms(user.DistinguishedName, acmList, "public", "")
	if !resp.Success {
		return NewAppError(http.StatusInternalServerError, err, "could not roll up zip file acms")
	}
	if !resp.AcmValid {
		return NewAppError(http.StatusInternalServerError, err, "invalid acm")
	}
	// Write the rollup, and extract its banner
	banner, err := acmExtractItem("banner", resp.AcmInfo.Acm)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "invalid acm")
	}

	// Begin writing the manifest (associate portion with individual files)
	header, err := zip.FileInfoHeader(newManifestInfo("classification_manifest.txt", time.Now()))
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "Unable to create file acm info header")
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "unable to create manifest")
	}

	// Make sure that the first line of the manifest is the rollup portion,
	// so that it's obvious what the overall classification is.
	// And write the portion and name of each individual file
	w.Write([]byte(fmt.Sprintf("%s\n\n", banner)))
	for _, m := range manifest.Files {
		w.Write([]byte(fmt.Sprintf("(%s) %s\n", m.Portion, path.Clean(m.Name))))
	}
	// And write the portion and name of each individual file
	w.Write([]byte(fmt.Sprintf("\n%s\n", banner)))

	return nil
}

// Get a reader on the ciphertext - locally if it exists, or range requested out of S3 otherwise
func zipReadCloser(dp ciphertext.CiphertextCache, logger *zap.Logger, rName ciphertext.FileId, totalLength int64) (io.ReadCloser, error) {
	// Range request it if we don't have it
	f, _, err := dp.NewPuller(logger, rName, totalLength, 0, -1)
	return f, err
}

// newFileInfo is everything that a zip needs to know about the file
func newFileInfo(obj models.ODObject, fqPath string) *zipFileInfo {
	return &zipFileInfo{
		size:    obj.ContentSize.Int64,
		name:    path.Clean(fqPath),
		modTime: obj.ModifiedDate,
		mode:    os.FileMode(0600),
		isDir:   false,
	}
}

// newManifestInfo is everything that a zip needs to know about the file
func newManifestInfo(fqPath string, dt time.Time) *zipFileInfo {
	return &zipFileInfo{
		name:    path.Clean(fqPath),
		modTime: dt,
		mode:    os.FileMode(0600),
		isDir:   false,
	}
}

// zipWriteFile writes filestreams to the archive. Security checks should already have taken place.
// Note that we do NOT support file paths now.  We only produce flat zips of individual files, with
// no original directory hierarchy.
func zipWriteFile(
	ctx context.Context,
	h AppServer,
	obj models.ODObject,
	zw *zip.Writer,
	userPermission models.ODObjectPermission,
	manifest *zipManifest,
) *AppError {
	totalLength := obj.ContentSize.Int64
	if totalLength <= 0 {
		logger.Debug("skipping object have no content length")
		return nil
	}
	logger := LoggerFromContext(ctx)
	dp := ciphertext.FindCiphertextCacheByObject(&obj)
	// Using captured permission, derive filekey
	var fileKey []byte
	fileKey = crypto.ApplyPassphrase(dp.GetMasterKey(), userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		return NewAppError(http.StatusInternalServerError, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object")
	}

	// Get the ciphertext for this file
	rName := ciphertext.FileId(obj.ContentConnector.String)
	cipherReader, err := zipReadCloser(dp, logger, rName, totalLength)
	if cipherReader != nil {
		defer cipherReader.Close()
	}
	if err != nil {
		logger.Error("unable to create puller for PermanentStorage", zap.Error(err))
		return NewAppError(http.StatusInternalServerError, err, "Unable to create pullet to read files")
	}
	logger.Debug("permanentstorage pull for zip begin", zap.String("fname", obj.Name), zap.Int64("bytes", totalLength))

	// Write the file header out, to properly set timestamps and permissions
	header, err := zip.FileInfoHeader(newFileInfo(obj, obj.Name))
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "unable to create file info header")
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, err, "unable to write zip file")
	}

	// Actually send back the cipherFile to zip stream - decrypted
	byteRange := &crypto.ByteRange{Start: 0, Stop: -1}
	var actualLength int64
	_, actualLength, err = h.Conf.EncryptableFunctions.DoCipherByReaderWriter(
		logger,
		cipherReader,
		w,
		fileKey,
		obj.EncryptIV,
		"zip for client",
		byteRange,
	)
	logger.Debug("file pull for zip end", zap.String("fname", obj.Name), zap.Int64("bytes", actualLength))
	return nil
}

func zipHasAccess(ctx context.Context, h AppServer, dbObject *models.ODObject) (bool, models.ODObjectPermission) {
	logger := LoggerFromContext(ctx)
	stillExists := dbObject.IsDeleted == false && dbObject.IsExpunged == false && dbObject.IsAncestorDeleted == false
	if !stillExists {
		logger.Info("object no longer exists for zip", zap.String("objectid", hex.EncodeToString(dbObject.ID)))
		return false, models.ODObjectPermission{}
	}
	caller, _ := CallerFromContext(ctx)
	aacAuth := auth.NewAACAuth(logger, h.AAC)
	if _, err := aacAuth.IsUserAuthorizedForACM(caller.DistinguishedName, dbObject.RawAcm.String); err != nil {
		logger.Info("auth error", zap.Error(err))
		return false, models.ODObjectPermission{}
	}
	return isUserAllowedToReadWithPermission(ctx, dbObject)
}

// zipIncludeFile puts a single file into the zipArchive.
func zipIncludeFile(
	ctx context.Context,
	h AppServer,
	obj models.ODObject,
	path string,
	zw *zip.Writer,
	manifest *zipManifest,
	usedNames *zipUsedNames,
) *AppError {
	hasAccess, userPermission := zipHasAccess(ctx, h, &obj)
	if hasAccess {
		if obj.RawAcm.Valid {
			portion, err := acmExtractItem("portion", obj.RawAcm.String)
			if err != nil {
				return NewAppError(http.StatusInternalServerError, err, "Could not get portion from object")
			}
			manifest.ACMs[obj.RawAcm.String] = true
			thisFile := zipManifestItem{
				Portion: portion,
				Name:    obj.Name,
			}
			manifest.Files = append(manifest.Files, thisFile)
			herr := zipWriteFile(ctx, h, obj, zw, userPermission, manifest)
			if herr != nil {
				return herr
			}
		} else {
			return NewAppError(http.StatusInternalServerError, nil, "No portion on object")
		}
	}
	return nil
}

func newManifest(zipSpec *protocol.Zip) *zipManifest {
	m := zipManifest{}
	m.ACMs = make(map[string]bool)
	return &m
}

func newUsedNames() *zipUsedNames {
	u := zipUsedNames{}
	u.UsedNames = make(map[string]bool)
	return &u
}

// If a name is going to conflict, then suggest one that doesn't.
// Notice that we can't just pass around a map[string]bool due to being modified during
// recursive searching.  So we opt to just have a pointer to a struct to modify, that might have
// more state later.
func zipSuggestName(u *zipUsedNames, name string) string {
	// No more directories, so use the base unconditionally
	name = path.Base(name)
	if u.UsedNames[name] {
		// Break up the file name to prepare for re-name
		ext := path.Ext(name)
		fname := name[0 : len(name)-len(ext)]
		// Search for a non-conflicting file name
		i := 1
		for {
			suggestedName := fmt.Sprintf("%s(%d)%s", fname, i, ext)
			if u.UsedNames[suggestedName] {
				i++
			} else {
				u.UsedNames[suggestedName] = true
				return suggestedName
			}
		}
	} else {
		u.UsedNames[name] = true
		return name
	}
}

// ZipSpecification says how you want the zip to happen
type ZipSpecification struct {
	ObjectIDs   []string `json:"objectIds"`
	FileName    string   `json:"fileName"`
	Disposition string   `json:"disposition"`
}

// zipRequestValidation applies validatin rules and sets fields on protocol.Zip request.
func zipRequestValidation(zipSpec *protocol.Zip) {
	if zipSpec.Disposition != "attachment" {
		zipSpec.Disposition = "inline"
	}
	// Make sure that the file is a proper file name without a path specifier, and .zip extension
	if len(zipSpec.FileName) > 0 && path.Ext(zipSpec.FileName) == ".zip" {
		// Make sure that we have a valid file base name, without being too opinionated about what it can be
		zipSpec.FileName = path.Clean(path.Base(zipSpec.FileName))
	} else {
		// Otherwise, just give it a default name
		zipSpec.FileName = "drive.zip"
	}
}

func (h AppServer) postZip(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "zip"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventExport")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "EXPORT")

	// Just give the zip file a standardized name for now.
	w.Header().Set("Content-Type", "application/zip")
	defer util.FinishBody(r.Body)
	var zipSpec protocol.Zip

	err := util.FullDecode(r.Body, &zipSpec)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "unable to perform zip due to malformed request")
		h.publishError(gem, herr)
		return herr
	}
	// This cleans up to make the input safe to go into the headers and sets defaults
	zipRequestValidation(&zipSpec)
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%s", zipSpec.Disposition, url.QueryEscape(zipSpec.FileName)))

	// Get started using the existing scheme from ShoeboxAPI
	// Using actual parameters allows for multi-select
	dao := DAOFromContext(ctx)
	logger := LoggerFromContext(ctx)

	// Start writing a zip file now
	manifest := newManifest(&zipSpec)
	zw := zip.NewWriter(w)
	usedNames := newUsedNames()
	zipSuggestName(usedNames, "classification_manifest.txt")

	// Remove duplicated object ids.
	uniqueIDs := make(map[string]bool)
	for _, v := range zipSpec.ObjectIDs {
		uniqueIDs[v] = true
	}
	for id := range uniqueIDs {
		// Get the root objects we need to zip into our file
		var err error
		var requestObject models.ODObject
		requestObject.ID, err = hex.DecodeString(id)
		if err != nil {
			herr := NewAppError(http.StatusBadRequest, err, "could not decode id")
			h.publishError(gem, herr)
			return herr
		}
		obj, err := dao.GetObject(requestObject, true)
		if err != nil {
			code, msg, err := getObjectDAOError(err)
			herr := NewAppError(code, err, msg)
			h.publishError(gem, herr)
			return herr
		}
		gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(obj))
		// Make sure that we don't have name collisions
		obj.Name = zipSuggestName(usedNames, obj.Name)
		if obj.ContentSize.Valid && obj.ContentSize.Int64 > 0 {
			// Go ahead and actually include this root object in the archive
			var herr *AppError
			herr = zipIncludeFile(ctx, h, obj, ".", zw, manifest, usedNames)
			if herr != nil {
				h.publishError(gem, herr)
				return herr
			}
		} else {
			logger.Info("zip not including file", zap.String("id", id))
		}
	}
	// We accumulated a lot of data in the manifest.  Write it out now.
	herr := zipWriteManifest(ctx, h.AAC, zw, manifest)
	if herr != nil {
		h.publishError(gem, herr)
		return herr
	}
	if err = zw.Close(); err != nil {
		herr := NewAppError(http.StatusInternalServerError, err, "could not properly close zip file")
		h.publishError(gem, herr)
		return herr
	}

	h.publishSuccess(gem, w)
	return nil
}

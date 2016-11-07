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

	"github.com/uber-go/zap"

	"encoding/hex"

	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/crypto"
	"decipher.com/object-drive-server/protocol"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
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
func zipWriteManifest(aacClient aac.AacService, ctx context.Context, logger zap.Logger, zw *zip.Writer, manifest *zipManifest) *AppError {
	user, ok := UserFromContext(ctx)
	if !ok {
		return NewAppError(500, nil, "unable to get user")
	}
	// manifest acms were stored in a list to make them unique
	// So now we need the list of unique values
	var acmList []string
	for a := range manifest.ACMs {
		acmList = append(acmList, a)
	}

	if len(acmList) == 0 {
		return NewAppError(400, nil, "No data to zip")
	}

	// Compute the rollup
	resp, err := aacClient.RollupAcms(user.DistinguishedName, acmList, "public", "")
	if !resp.Success {
		return NewAppError(500, err, "could not roll up zip file acms")
	}
	if !resp.AcmValid {
		return NewAppError(500, err, "invalid acm")
	}
	// Write the rollup, and extract its banner
	banner, err := acmExtractItem("banner", resp.AcmInfo.Acm)
	if err != nil {
		return NewAppError(500, err, "invalid acm")
	}

	// Begin writing the manifest (associate portion with individual files)
	header, err := zip.FileInfoHeader(newManifestInfo("classification_manifest.txt", time.Now()))
	if err != nil {
		return NewAppError(500, err, "Unable to create file acm info header")
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return NewAppError(500, err, "unable to create manifest")
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
func zipReadCloser(dp ciphertext.CiphertextCache, logger zap.Logger, rName ciphertext.FileId, totalLength int64) (io.ReadCloser, error) {
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
	h AppServer,
	ctx context.Context,
	obj models.ODObject,
	zw *zip.Writer,
	userPermission models.ODObjectPermission,
	manifest *zipManifest,
) *AppError {
	logger := LoggerFromContext(ctx)
	dp := ciphertext.FindCiphertextCacheByObject(&obj)
	// Using captured permission, derive filekey
	var fileKey []byte
	fileKey = crypto.ApplyPassphrase(h.MasterKey, userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		return NewAppError(500, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object")
	}

	// Get the ciphertext for this file
	rName := ciphertext.FileId(obj.ContentConnector.String)
	totalLength := obj.ContentSize.Int64
	cipherReader, err := zipReadCloser(dp, logger, rName, totalLength)
	if err != nil {
		logger.Error("unable to create puller for PermanentStorage", zap.String("err", err.Error()))
	}
	defer cipherReader.Close()
	logger.Debug("PermanentStorage pull for zip begin", zap.String("fname", obj.Name), zap.Int64("bytes", totalLength))

	// Write the file header out, to properly set timestamps and permissions
	header, err := zip.FileInfoHeader(newFileInfo(obj, obj.Name))
	if err != nil {
		return NewAppError(500, err, "Unable to create file info header")
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return NewAppError(500, err, "Unable to write zip file")
	}

	// Actually send back the cipherFile to zip stream - decrypted
	byteRange := &crypto.ByteRange{Start: 0, Stop: -1}
	var actualLength int64
	_, actualLength, err = crypto.DoCipherByReaderWriter(
		logger,
		cipherReader,
		w,
		fileKey,
		obj.EncryptIV,
		"zip for client",
		byteRange,
	)
	cipherReader.Close()
	logger.Debug("s3 pull for zip end", zap.String("fname", obj.Name), zap.Int64("bytes", actualLength))
	return nil
}

func zipHasAccess(h AppServer, ctx context.Context, obj *models.ODObject) (bool, models.ODObjectPermission) {
	logger := LoggerFromContext(ctx)
	stillExists := obj.IsDeleted == false && obj.IsExpunged == false && obj.IsAncestorDeleted == false
	if !stillExists {
		logger.Error("object no longer exists for zip")
		return false, models.ODObjectPermission{}
	}
	if err := h.isUserAllowedForObjectACM(ctx, obj); err != nil {
		if IsDeniedAccess(err) {
			logger.Error("zip does not give access", zap.String("err", err.Error()))
			return false, models.ODObjectPermission{}
		}
		logger.Error("zip unable to check acm for access", zap.String("err", err.Error()))
		return false, models.ODObjectPermission{}
	}
	return isUserAllowedToReadWithPermission(ctx, h.MasterKey, obj)
}

// zipIncludeFile puts a single file into the zipArchive.
func zipIncludeFile(
	h AppServer,
	ctx context.Context,
	obj models.ODObject,
	path string,
	zw *zip.Writer,
	manifest *zipManifest,
	usedNames *zipUsedNames,
) *AppError {
	hasAccess, userPermission := zipHasAccess(h, ctx, &obj)
	if hasAccess {
		if obj.RawAcm.Valid {
			portion, err := acmExtractItem("portion", obj.RawAcm.String)
			if err != nil {
				return NewAppError(500, err, "Could not get portion from object")
			}
			manifest.ACMs[obj.RawAcm.String] = true
			thisFile := zipManifestItem{
				Portion: portion,
				Name:    obj.Name,
			}
			manifest.Files = append(manifest.Files, thisFile)
			herr := zipWriteFile(h, ctx, obj, zw, userPermission, manifest)
			if herr != nil {
				return herr
			}
		} else {
			return NewAppError(500, nil, "No portion on object")
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
	// Just give the zip file a standardized name for now.
	w.Header().Set("Content-Type", "application/zip")
	defer util.FinishBody(r.Body)
	var zipSpec protocol.Zip

	err := util.FullDecode(r.Body, &zipSpec)
	if err != nil {
		return NewAppError(500, err, "unable to perform zip due to malformed request")
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
			return NewAppError(500, err, "could not decode id")
		}
		obj, err := dao.GetObject(requestObject, true)
		if err != nil {
			code, msg, err := getObjectDAOError(err)
			return NewAppError(code, err, msg)
		}
		// Make sure that we don't have name collisions
		obj.Name = zipSuggestName(usedNames, obj.Name)
		if obj.ContentSize.Valid && obj.ContentSize.Int64 > 0 {
			// Go ahead and actually include this root object in the archive
			var herr *AppError
			herr = zipIncludeFile(h, ctx, obj, ".", zw, manifest, usedNames)
			if herr != nil {
				return herr
			}
		} else {
			logger.Info("zip not including file", zap.String("id", id))
		}
	}
	// We accumulated a lot of data in the manifest.  Write it out now.
	herr := zipWriteManifest(h.AAC, ctx, logger, zw, manifest)
	if herr != nil {
		return herr
	}
	zw.Close()
	return nil
}

package server

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/uber-go/zap"

	"encoding/hex"
	"encoding/json"

	"decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/utils"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
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

// these are all the methods required to satisfy the FileInfo interface
// otherwise, the zip archive will have wrong dates and permissions when unpacked
func (z *zipFileInfo) Size() int64        { return z.size }
func (z *zipFileInfo) Name() string       { return z.name }
func (z *zipFileInfo) ModTime() time.Time { return z.modTime }
func (z *zipFileInfo) IsDir() bool        { return z.isDir }
func (z *zipFileInfo) Sys() interface{}   { return z.sys }
func (z *zipFileInfo) Mode() os.FileMode  { return z.mode }

// This is an item in the manifest
type zipManifestItem struct {
	Banner string
	Name   string
}

// All the state we need to write a manifest
// Eventually, we will need rollup info.
type zipManifest struct {
	Files []zipManifestItem
}

// All the state we need to deconflict names in a directory
type zipUsedNames struct {
	UsedNames map[string]bool
}

// We need to retain classifications of files, so we need to write something
// to re-associate them, short of doing file renaming to retain classification
func zipWriteManifest(logger zap.Logger, zw *zip.Writer, meta *zipManifest) *AppError {
	fqManifest := ".drive.manifest"
	header, err := zip.FileInfoHeader(newManifestInfo(fqManifest, time.Now()))
	if err != nil {
		return NewAppError(500, err, "Unable to create file acm info header")
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return NewAppError(500, err, "unable to create manifest")
	}
	for _, m := range meta.Files {
		line := fmt.Sprintf("%s:%s\n", m.Banner, path.Clean(m.Name))
		w.Write([]byte(line))
	}
	return nil
}

//Get a reader on the ciphertext - locally if it exists, or range requested out of S3 otherwise
func zipReadCloser(dp DrainProvider, logger zap.Logger, rName FileId, totalLength int64) (io.ReadCloser, error) {
	//Range request it if we don't have it
	cachedFileName := dp.Resolve(NewFileName(rName, ".cached"))
	fqFileName := dp.Files().Resolve(cachedFileName)
	if _, err := os.Stat(fqFileName); os.IsNotExist(err) {
		return dp.NewS3Puller(logger, rName, totalLength, 0, -1)
	}
	//We have it locally
	return dp.Files().Open(cachedFileName)
}

//newFileInfo is everything that a zip needs to know about the file
func newFileInfo(obj models.ODObject, fqPath string) *zipFileInfo {
	return &zipFileInfo{
		size:    obj.ContentSize.Int64,
		name:    path.Clean(fqPath),
		modTime: obj.ModifiedDate,
		mode:    os.FileMode(0600),
		isDir:   false,
	}
}

//newManifestInfo is everything that a zip needs to know about the file
func newManifestInfo(fqPath string, dt time.Time) *zipFileInfo {
	return &zipFileInfo{
		name:    path.Clean(fqPath),
		modTime: dt,
		mode:    os.FileMode(0600),
		isDir:   false,
	}
}

//newFileAcmInfo is everything that a zip needs to know about the file
func newFileAcmInfo(fqPath string, rawAcm string, dt time.Time) *zipFileInfo {
	return &zipFileInfo{
		size:    int64(len(rawAcm)),
		name:    path.Clean(fqPath),
		modTime: dt,
		mode:    os.FileMode(0600),
		isDir:   false,
	}
}

//newFolderInfo is everything needed to write a folder into the archive
func newFolderInfo(fqPath string, dt time.Time) *zipFileInfo {
	return &zipFileInfo{
		name:    path.Clean(fqPath),
		modTime: dt,
		mode:    os.FileMode(0700),
		isDir:   true,
	}
}

//Stream write to the archive.
//We should have already security checked this file before writing
func zipWriteFile(
	h AppServer,
	ctx context.Context,
	logger zap.Logger,
	dp DrainProvider,
	obj models.ODObject,
	zw *zip.Writer,
	fqName string,
	userPermission models.ODObjectPermission,
) *AppError {
	//Write an acm for this folder
	if obj.RawAcm.Valid {
		fqAcm := fqName + ".acm"
		header, err := zip.FileInfoHeader(newFileAcmInfo(fqAcm, obj.RawAcm.String, obj.ModifiedDate))
		if err != nil {
			return NewAppError(500, err, "Unable to create file acm info header")
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			return NewAppError(500, err, "Unable to write acm")
		}
		w.Write([]byte(obj.RawAcm.String))
	}

	// Using captured permission, derive filekey
	var fileKey []byte
	fileKey = utils.ApplyPassphrase(h.MasterKey, userPermission.PermissionIV, userPermission.EncryptKey)
	if len(fileKey) == 0 {
		return NewAppError(500, errors.New("Internal Server Error"), "Internal Server Error - Unable to derive file key from user permission to read/view this object")
	}

	// Get the ciphertext for this file
	rName := FileId(obj.ContentConnector.String)
	totalLength := obj.ContentSize.Int64
	cipherReader, err := zipReadCloser(dp, logger, rName, totalLength)
	if err != nil {
		logger.Error("unable to create puller for S3", zap.String("err", err.Error()))
	}
	defer cipherReader.Close()
	logger.Debug("s3 pull for zip begin", zap.String("fname", fqName), zap.Int64("bytes", totalLength))

	// Write the file header out, to properly set timestamps and permissions
	header, err := zip.FileInfoHeader(newFileInfo(obj, fqName))
	if err != nil {
		return NewAppError(500, err, "Unable to create file info header")
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return NewAppError(500, err, "Unable to write zip file")
	}

	//Actually send back the cipherFile to zip stream - decrypted
	byteRange := &utils.ByteRange{Start: 0, Stop: -1}
	var actualLength int64
	_, actualLength, err = utils.DoCipherByReaderWriter(
		logger,
		cipherReader,
		w,
		fileKey,
		obj.EncryptIV,
		"zip for client",
		byteRange,
	)
	cipherReader.Close()
	logger.Debug("s3 pull for zip end", zap.String("fname", fqName), zap.Int64("bytes", actualLength))
	return nil
}

//Get the security banner for this file
func zipExtractBanner(obj models.ODObject, logger zap.Logger) string {
	if obj.RawAcm.Valid {
		var acm interface{}
		acmBytes := []byte(obj.RawAcm.String)
		err := json.Unmarshal(acmBytes, &acm)
		if err == nil {
			acmData, acmDataOk := acm.(map[string]interface{})
			if acmDataOk {
				return acmData["banner"].(string)
			}
		} else {
			logger.Warn(
				"acm parse during zip",
				zap.String("err", err.Error()),
				zap.String("acm", obj.RawAcm.String),
			)
		}
	}
	return ""
}

// Put a single file into the zipArchive
func zipIncludeFile(
	h AppServer,
	ctx context.Context,
	logger zap.Logger,
	dp DrainProvider,
	dao dao.DAO,
	obj models.ODObject,
	path string,
	zw *zip.Writer,
	manifest *zipManifest,
) *AppError {
	hasAccess, userPermission := isUserAllowedToReadWithPermission(ctx, h.MasterKey, &obj)
	if hasAccess {
		banner := zipExtractBanner(obj, logger)
		fqName := obj.Name
		thisFile := zipManifestItem{
			Banner: banner,
			Name:   fqName,
		}
		manifest.Files = append(manifest.Files, thisFile)

		herr := zipWriteFile(h, ctx, logger, dp, obj, zw, fqName, userPermission)
		if herr != nil {
			return herr
		}
	}
	return nil
}

// Put this Folder into the zip file.
func zipIncludeFolder(
	h AppServer,
	ctx context.Context,
	logger zap.Logger,
	dp DrainProvider,
	dao dao.DAO,
	parentObject models.ODObject,
	fpath string,
	zw *zip.Writer,
	manifest *zipManifest,
) *AppError {
	var pagingRequest protocol.PagingRequest
	var err error

	//Page over the folder
	pagingRequest.PageNumber = 1
	pagingRequest.PageSize = 2000
	pagingRequest.ObjectID = hex.EncodeToString(parentObject.ID)
	fqPath := fmt.Sprintf("%s/%s", fpath, parentObject.Name)
	parentObject, err = assignObjectIDFromPagingRequest(&pagingRequest, parentObject)
	if err != nil {
		return NewAppError(400, err, "Object Identifier in Request URI is not a hex string")
	}

	//Write an acm for this folder
	if parentObject.RawAcm.Valid {
		fqAcm := fqPath + ".acm"
		header, err := zip.FileInfoHeader(newFileAcmInfo(fqAcm, parentObject.RawAcm.String, parentObject.ModifiedDate))
		if err != nil {
			return NewAppError(500, err, "Unable to create file acm info header")
		}

		w, err := zw.CreateHeader(header)
		if err != nil {
			return NewAppError(500, err, "Unable to write folder acm")
		}
		w.Write([]byte(parentObject.RawAcm.String))
	}

	//Page across the data, keeping track of used names for deconflict
	usedNames := newUsedNames()
	for {
		resultSet, err := dao.GetChildObjects(pagingRequest, parentObject)
		if err != nil {
			return NewAppError(500, err, "Problem fetching child objects")
		}
		objCount := len(resultSet.Objects)
		//Exit when we are at last page
		if objCount == 0 {
			break
		}
		//Iterate all the objects in this page
		for i := 0; i < objCount; i++ {
			obj := resultSet.Objects[i]
			isFolder := obj.TypeName.Valid && obj.TypeName.String == "Folder"
			obj.Name = zipSuggestName(usedNames, obj.Name, isFolder)

			// Check read permission, and capture permission for the encryptKey
			// Check if the user has permissions to read the ODObject
			//		Permission.grantee matches caller, and AllowRead is true
			hasAccess, userPermission := isUserAllowedToReadWithPermission(ctx, h.MasterKey, &obj)

			if hasAccess {
				banner := zipExtractBanner(obj, logger)
				//Record access level for all items - file or folder
				fqName := fmt.Sprintf("%s/%s", fqPath, obj.Name)
				thisFile := zipManifestItem{
					Banner: banner,
					Name:   fqName,
				}
				manifest.Files = append(manifest.Files, thisFile)

				//XXX This is an app-specific idea to have folders, without a way to efficiently
				// just ask for the children of a node without a full query
				//XXX also app specific as there is an assumption that Folders wont have streams to be zipped.
				if isFolder {
					herr := zipIncludeFolder(h, ctx, logger, dp, dao, obj, fqPath, zw, manifest)
					if herr != nil {
						return herr
					}
				} else {
					herr := zipWriteFile(h, ctx, logger, dp, obj, zw, fqName, userPermission)
					if herr != nil {
						return herr
					}
				}
			}
		}
		pagingRequest.PageNumber++
	}
	return nil
}

//This is a structure where we track the names that
//are already in use for this directory
func newUsedNames() *zipUsedNames {
	u := zipUsedNames{}
	u.UsedNames = make(map[string]bool)
	return &u
}

//If a name is going to conflict, then suggest one that doesn't.
//Notice that we can't just pass around a map[string]bool due to being modified during
//recursive searching.  So we opt to just have a pointer to a struct to modify, that might have
//more state later.
func zipSuggestName(u *zipUsedNames, name string, isFolder bool) string {
	if u.UsedNames[name] {
		i := 1
		var suggestedName string
		if isFolder {
			//Search for a non-conflicting folder name
			for {
				suggestedName = fmt.Sprintf("%s(%d)", name, i)
				if u.UsedNames[suggestedName] {
					i++
				} else {
					u.UsedNames[suggestedName] = true
					return suggestedName
				}
			}
		} else {
			//Break up the file name to prepare for re-name
			ext := path.Ext(name)
			var fname string
			withExtension := len(ext) > 0
			if withExtension {
				fname = name[:len(ext)-1]
			} else {
				fname = name
			}
			//Search for a non-conflicting file name
			for {

				if withExtension {
					suggestedName = fmt.Sprintf("%s(%d)%s", fname, i, ext)
				} else {
					suggestedName = fmt.Sprintf("%s(%d)", fname, i)
				}
				if u.UsedNames[suggestedName] {
					i++
				} else {
					u.UsedNames[suggestedName] = true
					return suggestedName
				}
			}
		}
	} else {
		u.UsedNames[name] = true
		return name
	}
}

//
// Note that when we zip files, we actually modify the name on the objects to de-conflict.
// We assume that the objects are not cached somewhere in the query, which would make such mods a problem.
//
func (h AppServer) getZip(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	//Just give the zip file a standardized name for now.
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=drive.zip")

	//Get started using the existing scheme from ShoeboxAPI
	//Using actual parameters allows for multi-select
	dao := DAOFromContext(ctx)
	dp := h.DrainProvider
	logger := LoggerFromContext(ctx)

	//Start writing a zip file now
	manifest := zipManifest{}
	zw := zip.NewWriter(w)
	usedNames := newUsedNames()
	for k, v := range r.URL.Query() {
		//object-drive-ui is inconsistent, and sometimes uses id for folderId in multi-select.
		//Workaround... once we have the ObjectID, we can ask the object what type it is.
		isID := k == "folderId" || k == "id"
		if isID {
			for _, id := range v {
				//Get the root objects we need to zip into our file
				var err error
				var requestObject models.ODObject
				requestObject.ID, err = hex.DecodeString(id)
				if err != nil {
					return NewAppError(500, err, "could not decode folderId")
				}
				obj, err := dao.GetObject(requestObject, true)
				if err != nil {
					code, msg := getObjectDAOError(err)
					return NewAppError(code, err, msg)
				}
				//Go ahead and actually include this root object in the archive
				var herr *AppError
				isFolder := obj.TypeName.Valid && obj.TypeName.String == "Folder"
				obj.Name = zipSuggestName(usedNames, obj.Name, isFolder)
				if isFolder {
					herr = zipIncludeFolder(h, ctx, logger, dp, dao, obj, ".", zw, &manifest)
				} else {
					herr = zipIncludeFile(h, ctx, logger, dp, dao, obj, ".", zw, &manifest)
				}
				if herr != nil {
					return herr
				}
			}
		}
	}
	//We accumulated a lot of data in the manifest.  Write it out now.
	herr := zipWriteManifest(logger, zw, &manifest)
	if herr != nil {
		return herr
	}
	zw.Close()
	return nil
}

package ciphertext

import (
	"io"
	"os"
	"time"

	"decipher.com/object-drive-server/metadata/models"

	"github.com/uber-go/zap"
)

const (
	//S3_DEFAULT_CIPHERTEXT_CACHE is the main ciphertext cache in use
	S3_DEFAULT_CIPHERTEXT_CACHE = CiphertextCacheName("S3_DEFAULT")
)

// CiphertextCacheName looks up ciphertext caches
type CiphertextCacheName string

// FileId is the raw random name with no extension
type FileId string

// FileName is the raw random name with no directory, and extension
type FileName string

// FileNameCached is the name prefixed with cache location, but not mount location
type FileNameCached string

// CiphertextCache handles the cached transfer of data in and out of permanent storage
type CiphertextCache interface {
	Files() FileSystem
	// This is the location where files get cached, and is used to organize things in the drain.  It's either an S3 or filesystem path off of CacheRoot()
	//Cache() string
	// Resolve these to locations in the drain provider, which doesn't say anything about where it is on filesystem - not fully qualified yet
	Resolve(FileName) FileNameCached
	// Writeback moves files from the cache into some kind of permanent storage (the drain)
	Writeback(rName FileId, size int64) error
	// NewPuller creates a virtual io.ReadCloser that pulls from PermanentStorage
	NewPuller(logger zap.Logger, rName FileId, totalLength, cipherStartAt, cipherStopAt int64) (io.ReadCloser, bool, error)
	// PermanentStorage handles reads and writes out of the cache
	GetPermanentStorage() PermanentStorage
	// CacheInventory gives a text listing of work outstanding before we can safely terminate
	CacheInventory(w io.Writer, verbose bool)
	// CountUploaded is a count of work items that need to complete before we can safely terminate
	CountUploaded() int
	// GetCiphertextCacheSelector is the key that this provider is stored under
	GetCiphertextCacheSelector() CiphertextCacheName
	// SetCiphertextCacheSelector is the key that we are going to store this under
	SetCiphertextCacheSelector(CiphertextCacheSelector CiphertextCacheName)
	// ReCache an object in the background
	BackgroundRecache(rName FileId, totalLength int64)
	// GetMasterKey is the key for this cache
	GetMasterKey() string
}

// ciphertextCaches is the named set of local caches that are bound to a remote bucket (S3 or possibly something else)
// This map is mutated in main on setup, and never edited after that
var ciphertextCaches = make(map[CiphertextCacheName]CiphertextCache)

// FindCiphertextCacheByObject gets us a drain provider that corresponds with the object
//
//  This implementation ASSUMES that main.go is setting us up with a propvider per selector
func FindCiphertextCacheByObject(obj *models.ODObject) CiphertextCache {
	//When we have an API token, and a way to configure multiple providers, we simply pick a provider as a functino object's properties (already tested to work)
	//For now, every object is getting default, but we can't change this without getting unique configs per CiphertextCache
	return FindCiphertextCache(S3_DEFAULT_CIPHERTEXT_CACHE)
}

// FindCiphertextCache gets us a drain provider by selector.  We ONLY use this to construct drain providers.  Ask for it by object otherwise.
func FindCiphertextCache(selector CiphertextCacheName) CiphertextCache {
	dp := ciphertextCaches[selector]
	if dp == nil {
		dp = ciphertextCaches[selector]
	}
	return dp
}

// FindCiphertextCacheList gets a list of known ciphertext caches
func FindCiphertextCacheList() []CiphertextCache {
	var answer []CiphertextCache
	for _, v := range ciphertextCaches {
		answer = append(answer, v)
	}
	return answer
}

// SetCiphertextCache sets an OD_CACHE_PARTITION (assuming multiple in the future) to a drain provider
// ONLY do this in single-threaded main setup, not while the system runs - so that we don't need to put RWMutexes around these
func SetCiphertextCache(selector CiphertextCacheName, dp CiphertextCache) {
	//Note that we use read locks everywhere else, and this should actually never be contended,
	//because setup of the set of ciphertext caches happens single-threaded in main on startup.
	ciphertextCaches[selector] = dp
}

// PermanentStorage is a generic type for mocking out or replacing S3
type PermanentStorage interface {
	Upload(fIn io.ReadSeeker, key *string) error
	Download(fOut io.WriterAt, key *string) (int64, error)
	GetStream(key *string, begin, end int64) (io.ReadCloser, error)
	GetName() *string
}

// CiphertextCacheFilesystemMountPoint is the mount point for CiphertextCache.CacheLocation()
// TODO how is this instantiated?
type CiphertextCacheFilesystemMountPoint struct {
	Root string
}

// FileSystem is an instance of "os" wrapped up to hide the implementation and location of the cache
type FileSystem interface {
	Resolve(fName FileNameCached) string
	Open(fName FileNameCached) (*os.File, error)
	Remove(fName FileNameCached) error
	Rename(fNameSrc, fNameDst FileNameCached) error
	Create(fName FileNameCached) (*os.File, error)
	Stat(fName FileNameCached) (os.FileInfo, error)
	MkdirAll(fName FileNameCached, perm os.FileMode) error
	RemoveAll(fName FileNameCached) error
	Chtimes(name FileNameCached, atime time.Time, mtime time.Time) error
}

// NewFileName turns an abstract id into a filename with an extension
func NewFileName(rName FileId, ext string) FileName {
	return FileName(string(rName) + ext)
}

// Chtimes touches the timestamp
func (c CiphertextCacheFilesystemMountPoint) Chtimes(name FileNameCached, atime time.Time, mtime time.Time) error {
	return os.Chtimes(c.Root+"/"+string(name), atime, mtime)
}

// Resolve the location relative to the mount point, which is required for debugging
func (c CiphertextCacheFilesystemMountPoint) Resolve(fName FileNameCached) string {
	return c.Root + "/" + string(fName)
}

// Open wraps os.Open for use with the cache
func (c CiphertextCacheFilesystemMountPoint) Open(fName FileNameCached) (*os.File, error) {
	return os.Open(c.Root + "/" + string(fName))
}

// Remove wraps os.Remove for use with the cache
func (c CiphertextCacheFilesystemMountPoint) Remove(fName FileNameCached) error {
	return os.Remove(c.Root + "/" + string(fName))
}

// Rename wraps os.Rename for use with the cache
func (c CiphertextCacheFilesystemMountPoint) Rename(fNameSrc, fNameDst FileNameCached) error {
	return os.Rename(c.Root+"/"+string(fNameSrc), c.Root+"/"+string(fNameDst))
}

// Create wraps os.Create for use with the cache
func (c CiphertextCacheFilesystemMountPoint) Create(fName FileNameCached) (*os.File, error) {
	return os.Create(c.Root + "/" + string(fName))
}

// Stat wraps os.Stat for use with the cache
func (c CiphertextCacheFilesystemMountPoint) Stat(fName FileNameCached) (os.FileInfo, error) {
	return os.Stat(c.Root + "/" + string(fName))
}

// MkdirAll wraps os.Mkdir for use with the cache
func (c CiphertextCacheFilesystemMountPoint) MkdirAll(fName FileNameCached, perm os.FileMode) error {
	return os.MkdirAll(c.Root+"/"+string(fName), perm)
}

// RemoveAll wraps os.RemoveAll for use with cache
func (c CiphertextCacheFilesystemMountPoint) RemoveAll(fName FileNameCached) error {
	return os.RemoveAll(c.Root + "/" + string(fName))
}

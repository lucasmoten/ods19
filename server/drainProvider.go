package server

import (
	"io"
	"os"
	"time"

	"decipher.com/object-drive-server/metadata/models"

	"github.com/uber-go/zap"
)

// FileId is the raw random name with no extension
type FileId string

// FileName is the raw random name with no directory, and extension
type FileName string

// FileNameCached is the name prefixed with cache location, but not mount location
type FileNameCached string

// CiphertextCache handles the cached transfer of data in and out of permanent storage
type CiphertextCache interface {
	Files() DrainCache
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
	GetCiphertextCacheSelector() string
	// SetCiphertextCacheSelector is the key that we are going to store this under
	SetCiphertextCacheSelector(CiphertextCacheSelector string)
	// ReCache an object in the background
	BackgroundRecache(rName FileId, totalLength int64)
}

// CiphertextCaches is the named set of local caches that are bound to a remote bucket (S3 or possibly something else)
var CiphertextCaches = make(map[string]CiphertextCache)

// this is the one mapped to "" and its real key
var defaultCiphertextCacheKey = "default"

// FindCiphertextCacheByObject gets us a drain provider that corresponds with the object
//
//  This implementation ASSUMES that main.go is setting us up with a propvider per key
func FindCiphertextCacheByObject(obj *models.ODObject) CiphertextCache {
	//When we have an API token, and a way to configure multiple providers, we simply pick a provider as a functino object's properties (already tested to work)
	return FindCiphertextCache(defaultCiphertextCacheKey)
}

// FindCiphertextCache gets us a drain provider by key.  We ONLY use this to construct drain providers.  Ask for it by object otherwise.
func FindCiphertextCache(key string) CiphertextCache {
	if key == "" {
		key = defaultCiphertextCacheKey
	}
	dp := CiphertextCaches[key]
	if dp == nil {
		key = defaultCiphertextCacheKey
		dp = CiphertextCaches[key]
	}
	return dp
}

// SetCiphertextCache sets an OD_CACHE_PARTITION (assuming multiple in the future) to a drain provider
func SetCiphertextCache(key string, dp CiphertextCache) {
	if key == "" {
		key = defaultCiphertextCacheKey
	}
	CiphertextCaches[key] = dp
	dp.SetCiphertextCacheSelector(key)
}

// SetDefaultCiphertextCache makes sure that if we don't specify which drain provider, we get the default
func SetDefaultCiphertextCache(key string) {
	defaultCiphertextCacheKey = key
}

// PermanentStorage is a generic type for mocking out or replacing S3
type PermanentStorage interface {
	Upload(fIn io.ReadSeeker, key *string) error
	Download(fOut io.WriterAt, key *string) (int64, error)
	GetObject(key *string, begin, end int64) (io.ReadCloser, error)
	GetBucket() *string
}

// DrainCacheData is the mount point for CiphertextCache.CacheLocation()
// TODO how is this instantiated?
type DrainCacheData struct {
	Root string
}

// DrainCache is an instance of "os" wrapped up to hide the implementation and location of the cache
type DrainCache interface {
	Resolve(fName FileNameCached) string
	Open(fName FileNameCached) (*os.File, error)
	Remove(fName FileNameCached) error
	Rename(fNameSrc, fNameDst FileNameCached) error
	Create(fName FileNameCached) (*os.File, error)
	Stat(fName FileNameCached) (os.FileInfo, error)
	MkdirAll(fName FileNameCached, perm os.FileMode) error
	RemoveAll(fName FileNameCached) error
	Chtimes(name FileNameCached, atime time.Time, mtime time.Time) error
	GetRoot() string
}

// NewFileName turns an abstract id into a filename with an extension
func NewFileName(rName FileId, ext string) FileName {
	return FileName(string(rName) + ext)
}

// Chtimes touches the timestamp
func (c DrainCacheData) Chtimes(name FileNameCached, atime time.Time, mtime time.Time) error {
	return os.Chtimes(c.Root+"/"+string(name), atime, mtime)
}

// Resolve the location relative to the mount point, which is required for debugging
func (c DrainCacheData) Resolve(fName FileNameCached) string {
	return c.Root + "/" + string(fName)
}

// Open wraps os.Open for use with the cache
func (c DrainCacheData) Open(fName FileNameCached) (*os.File, error) {
	return os.Open(c.Root + "/" + string(fName))
}

// Remove wraps os.Remove for use with the cache
func (c DrainCacheData) Remove(fName FileNameCached) error {
	return os.Remove(c.Root + "/" + string(fName))
}

// Rename wraps os.Rename for use with the cache
func (c DrainCacheData) Rename(fNameSrc, fNameDst FileNameCached) error {
	return os.Rename(c.Root+"/"+string(fNameSrc), c.Root+"/"+string(fNameDst))
}

// Create wraps os.Create for use with the cache
func (c DrainCacheData) Create(fName FileNameCached) (*os.File, error) {
	return os.Create(c.Root + "/" + string(fName))
}

// Stat wraps os.Stat for use with the cache
func (c DrainCacheData) Stat(fName FileNameCached) (os.FileInfo, error) {
	return os.Stat(c.Root + "/" + string(fName))
}

// MkdirAll wraps os.Mkdir for use with the cache
func (c DrainCacheData) MkdirAll(fName FileNameCached, perm os.FileMode) error {
	return os.MkdirAll(c.Root+"/"+string(fName), perm)
}

// RemoveAll wraps os.RemoveAll for use with cache
func (c DrainCacheData) RemoveAll(fName FileNameCached) error {
	return os.RemoveAll(c.Root + "/" + string(fName))
}

// GetRoot gives us the location where the cache is mounted in the filesystem
func (c DrainCacheData) GetRoot() string {
	return c.Root
}

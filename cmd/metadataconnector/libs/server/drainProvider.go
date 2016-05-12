package server

import (
	"os"
	"time"
)

// FileId is the raw random name with no extension
// These files DO NOT EXIST on filesystems
type FileId string

// FileName is the raw random name with no directory, and extension
type FileName string

// FileNameCached is the name prefixed with cache location, but not mount location
type FileNameCached string

// string is reserved for fully qualified paths

// DrainProvider handles the cached transfer of data in and out of permanent storage
type DrainProvider interface {
	Files() DrainCache
	// This is the location where files get cached, and is used to organize things in the drain.  It's either an S3 or filesystem path off of CacheRoot()
	//Cache() string
	// Resolve these to locations in the drain provider, which doesn't say anything about where it is on filesystem - not fully qualified yet
	Resolve(FileName) FileNameCached
	// CacheToDrain moves files from the cache into some kind of permanent storage (the drain)
	CacheToDrain(bucket *string, rName FileId, size int64) error
	// DrainToCache gets things back into the cache after they have gone into the drain
	DrainToCache(bucket *string, rName FileId) (*AppError, error)
}

// DrainCacheData is the mount point for DrainProvider.CacheLocation()
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

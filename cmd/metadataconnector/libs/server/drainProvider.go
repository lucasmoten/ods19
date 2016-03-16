package server

// DrainProvider handles the cached transfer of data in and out of permanent storage
type DrainProvider interface {
	// This is the location where files get cached, and is used to organize things in the drain
	CacheLocation() string
	// CacheToDrain moves files from the cache into some kind of permanent storage (the drain)
	CacheToDrain(bucket *string, rName string, size int64) error
	// DrainToCache gets things back into the cache after they have gone into the drain
	DrainToCache(bucket *string, rName string) (*AppError, error)
}

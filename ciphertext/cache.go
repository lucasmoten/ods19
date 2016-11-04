package ciphertext

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"syscall"

	globalconfig "decipher.com/object-drive-server/config"

	"github.com/uber-go/zap"
)

const (
	// PurgeAnomaly error code given when we purged something that wasn't cleaned up
	PurgeAnomaly = 1500
	// FailPurgeAnomaly error code given when we failed to purge something that wasn't cleaned up
	FailPurgeAnomaly = 1501
	// FailCacheWalk error code given when we tried to walk cache, and something went wrong
	FailCacheWalk = 1502
	// FailWriteback error code given when we could not cache to drain
	FailWriteback = 1504
)

// CiphertextCacheData moves data from cache to the drain...
type CiphertextCacheData struct {
	//ChunkSize is the size of blocks to pull from PermanentStorage
	ChunkSize int64
	//The key that this CiphertextCache is stored under
	CiphertextCacheSelector CiphertextCacheName
	//Where the CacheLocation is rooted on disk (ie: a very large drive mounted)
	files FileSystem

	//This is the place to write back persistence
	PermanentStorage PermanentStorage

	//Location of the cache
	CacheLocationString string

	//Dont begin purging anything until we are at this fraction of disk for cache
	//TODO: may need to tune on small systems where the OS is counted in the partition,
	// and it is a significant fraction of the disk.
	lowWatermark float64

	//Keep things in cache for a few minutes minimum, then delete based on value
	//TODO: may need to tune on small systems where the OS is counted in the partition,
	// and it is a significant fraction of the disk.
	ageEligibleForEviction int64

	//If we get to the high watermark, just start deleting until we get under it.
	//Note that if in the time period ageEligibleForEviction you upload enough
	//to stay at the highWatermark, you won't be able to stay within your cache limits.
	//TODO: may need to tune on small systems where the OS is counted in the partition,
	// and it is a significant fraction of the disk.
	highWatermark float64

	//The time to wait to walk the files
	walkSleep time.Duration

	//Logger for logging
	Logger zap.Logger

	// MasterKey is the secret passphrase used in scrambling keys
	MasterKey string
}

// NewCiphertextCacheRaw is a cache that goes off to PermanentStorage.
// Strategy:
//  * for uploads, *.uploaded files must make it to permanent storage eventually.
//  * for downloads, try disk first, then try permanent storage once.
//    if it is not in permanent storage, then it must be in one of our peers - who
//    is still trying to get it into permanent storage, so try
//    the peers once before resorting to re-checking permanent storage with sleep time in between.
//
//  When using this download rule, we should *never* be stalling for ciphertext
//  unless something is wrong.  Whether from PermanentStorage or a peer, the whole file
//  should be available once the database object exists.  In other words, it's basically a bug to ever see:
//
//    "unable to download out of PermanentStorage"
//
//  in the logs
//
//
//
func NewCiphertextCacheRaw(
	root string,
	name string,
	lowWatermark float64,
	highWatermark float64,
	ageEligibleForEviction int64,
	walkSleep time.Duration,
	chunkSize int64,
	logger zap.Logger,
	permanentStorage PermanentStorage,
	masterKey string) *CiphertextCacheData {
	d := &CiphertextCacheData{
		CiphertextCacheSelector: S3_DEFAULT_CIPHERTEXT_CACHE, //This will be overwritten when it is put into a map of caches
		PermanentStorage:        permanentStorage,
		files:                   CiphertextCacheFilesystemMountPoint{root},
		CacheLocationString:     name,
		lowWatermark:            lowWatermark,
		ageEligibleForEviction:  ageEligibleForEviction,
		highWatermark:           highWatermark,
		walkSleep:               walkSleep,
		ChunkSize:               chunkSize,
		Logger:                  logger,
		MasterKey:               masterKey,
	}
	CacheMustExist(d, logger)
	logger.Info("cache purge",
		zap.Float64("lowwatermark", lowWatermark),
		zap.Float64("highwatermark", highWatermark),
		zap.Int64("ageeligibleforeviction", ageEligibleForEviction),
		zap.Duration("walksleep", walkSleep),
	)
	return d
}

// GetMasterKey is the key for this cache - no more system global masterkey
// This means that in order to have a key, you need to have an object that it refers to
func (d *CiphertextCacheData) GetMasterKey() string {
	return d.MasterKey
}

// Resolve a name to somewhere in the cache, given the rName
func (d *CiphertextCacheData) Resolve(fName FileName) FileNameCached {
	return FileNameCached(d.CacheLocationString + "/" + string(fName))
}

// Files is the mount point of instances
func (d *CiphertextCacheData) Files() FileSystem {
	return d.files
}

// DrainUploadedFilesToSafetyRaw is the drain without the goroutine at the end
func (d *CiphertextCacheData) DrainUploadedFilesToSafetyRaw() {
	if d.PermanentStorage == nil {
		d.Logger.Info("PersistentStorage not used")
		return
	}
	//Walk through the cache, and handle .uploaded files
	fqCache := d.Files().Resolve(d.Resolve(""))
	err := filepath.Walk(
		fqCache,
		// We need to capture d because this interface won't let us pass it
		func(fqName string, f os.FileInfo, err error) (errReturn error) {
			if err != nil {
				d.Logger.Error(
					"error walking directory",
					zap.String("filename", fqName),
					zap.String("err", err.Error()),
				)
				// I didn't generate this error, so I am assuming that I can just log the problem.
				// TODO: this error is not being counted
				return nil
			}

			if f.IsDir() {
				return nil
			}
			size := f.Size()
			ext := path.Ext(fqName)
			if ext == ".uploaded" {
				fBase := path.Base(fqName)
				rName := FileId(fBase[:len(fBase)-len(ext)])
				err := d.Writeback(rName, size)
				if err != nil {
					d.Logger.Error("error draining cache", zap.String("err", err.Error()))
				}
			}
			if ext == ".caching" || ext == ".uploading" {
				d.Logger.Info("removing", zap.String("filename", fqName))
				//On startup, any .caching files are associated with now-dead goroutines.
				//On startup, any .uploading files are associated with now-dead uploads.
				os.Remove(fqName)
			}
			return nil
		},
	)
	if err != nil {
		d.Logger.Error("Unable to walk cache", zap.String("err", err.Error()))
	}
}

// DrainUploadedFilesToSafety moves files that were not completely sent to PermanentStorage yet, so that the instance is disposable.
// This can happen if the server reboots.
func (d *CiphertextCacheData) DrainUploadedFilesToSafety() {
	d.DrainUploadedFilesToSafetyRaw()
	d.Logger.Info("cache purge start")
	//Only now can we start to purge files
	go d.CachePurge()
}

func toKey(s string) *string {
	return &s
}

// Writeback drains to PermanentStorage.  Note that because this is async with respect to the http session,
// we cant return AppError.
//
// Dont delete the file here if something goes wrong... because the caller tries this multiple times
//
func (d *CiphertextCacheData) Writeback(rName FileId, size int64) error {
	outFileUploaded := d.Resolve(FileName(rName + ".uploaded"))
	key := toKey(string(d.Resolve(NewFileName(rName, ""))))

	//Get a filehandle to read the file to write back to permanent storage
	fIn, err := d.Files().Open(outFileUploaded)
	if err != nil {
		d.Logger.Error(
			"Cant writeback file",
			zap.String("filename", d.Files().Resolve(outFileUploaded)),
			zap.String("err", err.Error()),
		)
		return err
	}
	defer fIn.Close()

	if d.PermanentStorage != nil {
		d.Logger.Info(
			"writeback to PermanentStorage",
			zap.String("bucket", *d.PermanentStorage.GetName()),
			zap.String("key", *key),
		)

		err = d.PermanentStorage.Upload(fIn, key)
		if err != nil {
			d.Logger.Error(
				"Could not write to PermanentStorage",
				zap.String("err", err.Error()),
			)
			return err
		}
	}

	//Rename the file to note success
	outFileCached := d.Resolve(NewFileName(rName, ".cached"))
	err = d.Files().Rename(outFileUploaded, outFileCached)
	if err != nil {
		d.Logger.Error(
			"Unable to rename",
			zap.String("from", d.Files().Resolve(outFileUploaded)),
			zap.String("to", d.Files().Resolve(outFileCached)),
			zap.String("err", err.Error()),
		)
		return err
	}
	if d.PermanentStorage != nil {
		d.Logger.Info(
			"PermanentStorage stored",
			zap.String("rname", string(rName)),
		)
	}

	return err
}

func (d *CiphertextCacheData) doDownloadFromPermanentStorage(foutCaching FileNameCached, key *string) error {
	if d.PermanentStorage == nil {
		return fmt.Errorf("there is no PermanentStorage set")
	}

	//Do a whole file download from PermanentStorage
	fOut, err := d.Files().Create(foutCaching)
	defer fOut.Close()
	if err != nil {
		msg := "unable to write local buffer"
		d.Logger.Error(
			msg,
			zap.String("filename", d.Files().Resolve(foutCaching)),
			zap.String("err", err.Error()),
		)
		return err
	}
	_, err = d.PermanentStorage.Download(fOut, key)
	return err
}

// BackgroundRecache deals with the case where we go to retrieve a file, and we want to
// make a better effort than to throw an exception because it is not cached in our local cache.
// if another routine is caching, then wait for that to finish.
// if nobody is caching it, then we start that process.
func (d *CiphertextCacheData) BackgroundRecache(rName FileId, totalLength int64) {

	logger := d.Logger
	cachingPath := d.Resolve(NewFileName(rName, ".caching"))
	cachedPath := d.Resolve(NewFileName(rName, ".cached"))

	logger.Info(
		"caching file",
		zap.String("filename", string(cachedPath)),
	)

	if _, err := d.Files().Stat(cachingPath); os.IsNotExist(err) {
		// Start caching the file because this is not happening already.
		err = d.Recache(rName)
		if err != nil {
			logger.Error(
				"background recache failed",
				zap.String("err", err.Error()),
			)
			return
		}
	}
	logger.Info(
		"background recache done",
		zap.String("filename", string(cachedPath)),
	)
}

// Recache gets a WHOLE file back out of the drain into the cache.
func (d *CiphertextCacheData) Recache(rName FileId) error {

	// If it's already cached, then we have no work to do
	foutCached := d.Resolve(NewFileName(rName, ".cached"))
	if _, err := d.Files().Stat(foutCached); os.IsExist(err) {
		return nil
	}

	// We are not supposed to be trying to get multiple copies of the same ciphertext into cache at same time
	foutCaching := d.Resolve(NewFileName(rName, ".caching"))
	if _, err := d.Files().Stat(foutCaching); os.IsExist(err) {
		return err
	}

	if d.PermanentStorage != nil {
		d.Logger.Info(
			"recache from PermanentStorage",
			zap.String("key", string(rName)),
		)
	}

	key := toKey(string(d.Resolve(NewFileName(rName, ""))))

	// This file must ONLY exist for the duration of this function.
	// we must remove it or rename it before we exit.
	// It is used to lock downloads of this file.
	//
	// This is also why we must delete all caching files on startup.
	defer d.Files().Remove(foutCaching)

	var err error
	var fOut io.WriteCloser

	err = d.doDownloadFromPermanentStorage(foutCaching, key)
	if err != nil {
		d.Logger.Warn("download from PermanentStorage error", zap.String("err", err.Error()))
		// Check p2p.... it has to be there...
		var filep2p io.ReadCloser
		filep2p, err = useP2PFile(d.Logger, d.CiphertextCacheSelector, rName, 0)
		if err != nil {
			d.Logger.Error("p2p cannot find", zap.String("err", err.Error()))
		}
		if filep2p != nil {
			defer filep2p.Close()
			fOut, err = d.Files().Create(foutCaching)
			if err == nil {
				// We need to copy the *whole* file in this case.
				_, err = io.Copy(fOut, filep2p)
				fOut.Close()
				if err != nil {
					d.Logger.Error("p2p recache failed", zap.String("err", err.Error()))
				} else {
					d.Logger.Info("p2p recache success")
				}
				// leave err where it is.
			}
		}
	}

	// This only exists for exotic corner cases.  Without network errors,
	// this block should be unreachable.
	tries := 22
	waitTime := 1 * time.Second
	prevWaitTime := 0 * time.Second
	for tries > 0 && err != nil && d.PermanentStorage != nil {
		err = d.doDownloadFromPermanentStorage(foutCaching, key)
		tries--
		if err == nil {
			break
		} else {
			d.Logger.Error(
				"unable to download out of PermanentStorage or p2p",
				zap.Duration("seconds", waitTime/(time.Second)),
				zap.Int("more tries", tries-1),
				zap.String("key", string(rName)),
			)
			// Without a file length, this is our best guess
			time.Sleep(waitTime)
			// Fibonacci progression 1 1 2 3 ... ... 22 of them gives a total wait time of about 2 mins, or almost 8GB
			oldWaitTime := waitTime
			waitTime = prevWaitTime + waitTime
			prevWaitTime = oldWaitTime
		}
	}

	if err != nil {
		d.Logger.Error("giving up on recaching file", zap.String("err", err.Error()))
		return err
	}

	// Signal that we finally cached the file
	err = d.Files().Rename(foutCaching, foutCached)
	if err != nil {
		d.Logger.Error(
			"rename fail",
			zap.String("from", d.Files().Resolve(foutCaching)),
			zap.String("to", d.Files().Resolve(foutCached)),
		)
		return err
	}
	d.Logger.Info("fetched ciphertext", zap.String("rname", string(rName)))
	return nil
}

// CacheMustExist ensures that the cache directory exists.
func CacheMustExist(d CiphertextCache, logger zap.Logger) (err error) {
	if _, err = d.Files().Stat(d.Resolve("")); os.IsNotExist(err) {
		err = d.Files().MkdirAll(d.Resolve(""), 0700)
		cacheResolved := d.Files().Resolve(d.Resolve(""))
		logger.Info(
			"creating cache",
			zap.String("filename", cacheResolved),
		)
		if err != nil {
			return err
		}
	}
	return err
}

// filePurgeVisit visits every file in the cache to see if we should delete it.
//
// The whole point of this is to keep the cache as large as possible without
// filling up the disk, while maximizing the hit rate.  This means that we must
// estimate what is going to be a hit (lower age since last touched), and if
// a file is 10x larger its hit is worth 10x as much because it costs 10x as much to get it.
// The file size matters in the decision to remove files, but the age since last use
// matters much more.  We are expecting the getObjectStreamX to timestamp this file
// every time it's downloaded to ensure that we correctly value the file.
//
// Behavior:
//  disk usage below lowWatermark: ignore this file
//  disk usage above highWatermark: delete this file if it's old enough for eviction
//  disk usage in range of watermarks:
//      if file is too young to evict at all: ignore this file
//      if int(size/(age*age)) == 0: delete this file
//
//  The net effect is that some files remain young because they were recently uploaded or fetched.
//  Multiplying times filesize recognizes that the penalty for deleting a large file is proportional
//  to its size.  But because 1/(age*age) drops rapidly, even very large files will quickly become
//  eligible for deletion if not used.  If there are many files that are accessed often, then the large files
//  will be selected for deletion.  Large files that keep getting used will stay in cache
//  as long as they keep getting used.  Because they take N times longer to get stamped due to
//  the length of the file transfer, it is fair to make it take N times longer to get evicted.
//
//  The graph will look like a sawtooth between lowWatermark and highWatermark, where there is
//  a delay in size drops that is dependent on size and doubly dependent on age since last access.
//  Size and Age prioritize what is still sitting in cache when we hit lowWatermark.
//
func filePurgeVisit(d *CiphertextCacheData, fqName string, f os.FileInfo, err error) (errReturn error) {
	if err != nil {
		d.Logger.Error(
			"error walking directory",
			zap.String("filename", fqName),
			zap.String("err", err.Error()),
		)
		// I didn't generate this error, so I am assuming that I can just log the problem.
		// TODO: this error is not being counted
		return nil
	}

	//Ignore directories.  We should not have an unbounded number of directories.
	//And we must ignore h.CacheLocation
	if f.IsDir() {
		return nil
	}

	//Size and age determine the value of the file
	t := f.ModTime().Unix() //In units of second
	n := time.Now().Unix()  //In unites of second
	ageInSeconds := n - t
	size := f.Size()
	ext := path.Ext(string(fqName))

	//Get the current disk space usage
	sfs := syscall.Statfs_t{}
	err = syscall.Statfs(fqName, &sfs)
	if err != nil {
		d.Logger.Error(
			"unable to purge on statfs fail",
			zap.String("filename", fqName),
			zap.String("err", err.Error()),
		)
		return nil
	}
	//Fraction of disk used
	usage := 1.0 - float64(sfs.Bavail)/float64(sfs.Blocks)
	switch {
	//Note that .cached files are securely stored in S3 already
	case ext == ".cached":
		//If we hit usage high watermark, we essentially panic and start deleting from the cache
		//until we are at low watermark
		oldEnoughToEvict := (ageInSeconds > d.ageEligibleForEviction)
		fullEnoughToEvict := (usage > d.lowWatermark)
		mustEvict := (usage > d.highWatermark && ageInSeconds >= d.ageEligibleForEviction)
		// expect usage to sawtooth between lowWatermark and highWatermark
		// with the value of the file setting priority until we hit highWatermark
		if (oldEnoughToEvict && fullEnoughToEvict) || mustEvict {
			value := size / (ageInSeconds * ageInSeconds)
			if value == 0 || mustEvict {
				//Name is fully qualified, so use os call!
				errReturn := os.Remove(fqName)
				if errReturn != nil {
					d.Logger.Error(
						"unable to purge",
						zap.String("filename", fqName),
						zap.String("err", errReturn.Error()),
					)
					return nil
				}
				d.Logger.Info(
					"purge",
					zap.String("filename", fqName),
					zap.Int64("ageinseconds", ageInSeconds),
					zap.Int64("size", size),
					zap.Float64("usage", usage),
				)
			}
		}
	default:
		//If something has been here for a week, and it's not cached, then it's
		//garbage.  If a machine has been turned off for a few days, the files
		//might legitimately be awaiting upload.  Other states are certainly
		//garbage after only a few hours.
		if ageInSeconds > 60*60*24*7 {
			errReturn := os.Remove(fqName)
			if errReturn != nil {
				d.Logger.Error(
					"unable to purge",
					zap.String("filename", fqName),
					zap.String("err", errReturn.Error()),
				)
				return nil
			}
			//Count this anomaly
			d.Logger.Warn(
				"purged for age",
				zap.String("filename", fqName),
				zap.Int64("age", ageInSeconds),
				zap.Float64("usage", usage),
			)
			return nil
		}
	}
	return
}

// CacheInventory writes an inventory of what's in the cache to a writer for the stats page
func (d *CiphertextCacheData) CacheInventory(w io.Writer, verbose bool) {
	fqCache := d.Files().Resolve(d.Resolve(""))
	fmt.Fprintf(w, "\n\ncache at %s on %s\n", fqCache, globalconfig.NodeID)
	filepath.Walk(
		fqCache,
		func(name string, f os.FileInfo, err error) error {
			if err == nil && strings.Compare(name, fqCache) != 0 {
				if verbose || strings.HasSuffix(name, ".uploaded") {
					fmt.Fprintf(w, "%s\n", name)
				}
			}
			return nil
		},
	)
}

// CountUploaded tells us how many files still need to be uploaded, which we use
// to determine if this is a good time to shut down
func (d *CiphertextCacheData) CountUploaded() int {
	uploaded := 0
	fqCache := d.Files().Resolve(d.Resolve(""))
	filepath.Walk(
		fqCache,
		func(name string, f os.FileInfo, err error) error {
			if err == nil && strings.Compare(name, fqCache) != 0 {
				if strings.HasSuffix(name, ".uploaded") {
					uploaded++
				}
			}
			return nil
		},
	)
	return uploaded
}

// CachePurge will periodically delete files that do not need to be in the cache.
func (d *CiphertextCacheData) CachePurge() {
	if d.PermanentStorage == nil {
		d.Logger.Info("PersistentStorage is nil.  purge is disabled.")
		return
	}
	// read from environment variables:
	//    lowWatermark (floating point 0..1)
	//    highWatermark (floating point 0..1)
	//    ageEligibleForEviction (integer seconds)
	var err error
	for {
		fqCache := d.Files().Resolve(d.Resolve(""))
		err = filepath.Walk(
			fqCache,
			func(name string, f os.FileInfo, err error) (errReturn error) {
				return filePurgeVisit(d, name, f, err)
			},
		)
		if err != nil {
			d.Logger.Error(
				"unable to walk cache",
				zap.String("filename", fqCache),
				zap.String("err", err.Error()),
			)
		}
		time.Sleep(d.walkSleep)
	}
}

// GetPermanentStorage gets the permanent storage provider plugged into this cache
func (d *CiphertextCacheData) GetPermanentStorage() PermanentStorage {
	return d.PermanentStorage
}

// GetCiphertextCacheSelector is the key that this is stored under
func (d *CiphertextCacheData) GetCiphertextCacheSelector() CiphertextCacheName {
	return d.CiphertextCacheSelector
}

// SetCiphertextCacheSelector sets the key by which we actually do the lookup
func (d *CiphertextCacheData) SetCiphertextCacheSelector(selector CiphertextCacheName) {
	d.CiphertextCacheSelector = selector
}

package ciphertext

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/util"

	"crypto/sha256"

	"encoding/hex"

	"io/ioutil"

	"go.uber.org/zap"
)

const FileStateCached = ".cached"
const FileStateCaching = ".caching"
const FileStateUploaded = ".uploaded"
const FileStateUploading = ".uploading"
const FileStateOrphaned = ".orphaned"

// CiphertextCacheData moves data from cache to the drain.
type CiphertextCacheData struct {
	// ChunkSize is the size of blocks to pull from PermanentStorage
	ChunkSize int64
	// CiphertextCacheZone is an identifier of the type of cache that CiphertextCache is stored under
	CiphertextCacheZone CiphertextCacheZone
	// files represents the root mount point of the cache location on disk (e.g. /cacheroot)
	files FileSystem
	// PermanentStorage is the place to write back persistence
	PermanentStorage PermanentStorage
	// CacheLocationString is a subfolder underneath the root comprising partition and database identifier
	CacheLocationString string
	// lowThresholdPercent denotes the lower threshold fraction where purging should be considered unless
	// the fileLimit is set
	lowThresholdPercent float64
	// ageEligibleForEviction indicates how long, in seconds, cached files should remain before eligible
	ageEligibleForEviction int64
	// highTresholdPercent denotes the upper threshold fraction where purging must occur more aggressively
	highThresholdPercent float64
	// fileLimit denotes max number of files to keep in cache. A value <= 0 is unlimited, the default
	fileLimit int64
	// walkSleep is the time to wait after a cache purge iteration before checking again
	walkSleep time.Duration
	// fileSleep is the time to wait between each file is checked
	fileSleep time.Duration
	// Logger for logging
	Logger *zap.Logger
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
// The *util.Loggable is a valid error type.  If it isn't nil, the server should NOT write to this cache.
// Generally, this means that the server should panic.
//
func NewCiphertextCacheRaw(
	zone CiphertextCacheZone,
	conf *config.DiskCacheOpts,
	dbID string,
	logger *zap.Logger,
	permanentStorage PermanentStorage,
) (*CiphertextCacheData, *util.Loggable) {
	//Do the unit conversions HERE
	d := &CiphertextCacheData{
		CiphertextCacheZone:    zone,
		PermanentStorage:       permanentStorage,
		files:                  CiphertextCacheFilesystemMountPoint{conf.Root},
		CacheLocationString:    filepath.Join(conf.Partition, dbID),
		lowThresholdPercent:    conf.LowThresholdPercent,
		ageEligibleForEviction: conf.EvictAge,
		highThresholdPercent:   conf.HighThresholdPercent,
		walkSleep:              time.Duration(conf.WalkSleep) * time.Second,
		ChunkSize:              conf.ChunkSize * 1024 * 1024,
		Logger:                 logger,
		MasterKey:              conf.MasterKey,
		fileLimit:              conf.FileLimit,
		fileSleep:              time.Duration(conf.FileSleep) * time.Millisecond,
	}
	CacheMustExist(d, logger)

	logger.Info("ciphertextcache created",
		zap.String("mount", conf.Root),
		zap.String("location", d.CacheLocationString),
	)
	return d, d.masterKeyCheck()
}

// Delete the local cache
func (d *CiphertextCacheData) Delete() error {
	return d.Files().RemoveAll(FileNameCached(d.CacheLocationString))
}

// haveCanary looks for the canary - we might not have one.  that is no cause for panic.
func (d *CiphertextCacheData) haveCanary(rName FileId) (string, *util.Loggable) {
	err := d.Recache(rName)
	if err != nil {
		// If it's just a key not found sentinel value, then just return that
		if err.Error() == PermanentStorageNotFoundErrorString {
			return "", err.(*util.Loggable)
		}
		return "", util.NewLoggable("ciphertextcache expected check fail", err)
	}
	rNameCached := NewFileName(rName, FileStateCached)
	nameCached := d.Files().Resolve(d.Resolve(rNameCached))
	_, err = os.Stat(nameCached)

	if err != nil {
		return "", util.NewLoggable("ciphertextcache stat error", err)
	}

	// Check the value to see if it matches expected
	f, err := os.Open(nameCached)
	if err != nil {
		return "", util.NewLoggable("ciphertextcache expected open fail", err)
	}
	defer f.Close()
	haveBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", util.NewLoggable("ciphertextcache expected read fail", err)
	}
	return string(haveBytes), nil
}

// expectCanary specifies which canary we expect, and writes it back so that it makes it to PermanentStorage
func (d *CiphertextCacheData) expectCanary(rName FileId, expected string) *util.Loggable {
	nameUploaded := d.Files().Resolve(d.Resolve(NewFileName(rName, FileStateUploaded)))
	defer os.Remove(nameUploaded)
	// Create the expected canary to write back.
	f, err := os.Create(nameUploaded)
	if err != nil {
		return util.NewLoggable("ciphertextcache expected write", err)
	}
	f.Write([]byte(expected))
	f.Close()
	// After writeback, it should get renamed to cached state
	err = d.Writeback(rName, int64(len(expected)))
	if err != nil {
		return util.NewLoggable("ciphertextcache expected writeback fail", err)
	}
	return nil
}

// masterKeyCheck will advise the caller to panic if the masterKey being used is definitely wrong
func (d *CiphertextCacheData) masterKeyCheck() *util.Loggable {
	// The expected value is a hex hash of our key stored in a canary file
	hashedKeyBytes := sha256.Sum256([]byte(d.MasterKey))
	expected := hex.EncodeToString(hashedKeyBytes[:])
	// The canary file
	rName := FileId("canary")
	have, err := d.haveCanary(rName)

	if have == "" {
		return d.expectCanary(rName, expected)
	}

	// If we were unable to get a canary (we could be the first one to try), then log it and say what canary we expect
	if err != nil {
		if err.Msg == PermanentStorageNotFoundErrorString {
			// This is a sentinel value that says that it's not set - not found is not an error
			d.Logger.Info("ciphertextcache canary is being set", zap.String("expectCanary", expected))
			return d.expectCanary(rName, expected)
		}
		return err
	}

	// Fail if we don't have what we expected and we have something specific
	if have != expected {
		// If we are going to fail to come up, delete the cached key, as it's invalid.
		rNameCached := NewFileName(rName, FileStateCached)
		nameCached := d.Files().Resolve(d.Resolve(rNameCached))
		os.Remove(nameCached)
		return util.NewLoggable("ciphertextcache canary mismatch", nil,
			zap.String("detail",
				"Other cluster members are using different values for OD_ENCRYPT_MASTERKEY or OD_ENCRYPT_ENABLED. Check your configuration settings.",
			),
			zap.String("haveCanary", have),
			zap.String("expectCanary", expected),
		)
	}

	// Let the logs know that we got a positive match on the canary
	d.Logger.Info("ciphertextcache canary is a positive match")
	return nil
}

// GetMasterKey is the key for this cache - no more system global masterkey
// This means that in order to have a key, you need to have an object that it refers to
func (d *CiphertextCacheData) GetMasterKey() string {
	return d.MasterKey
}

// Resolve a name to somewhere in the cache, given the rName
func (d *CiphertextCacheData) Resolve(fName FileName) FileNameCached {
	return FileNameCached(filepath.Join(d.CacheLocationString, string(fName)))
}

// Files is the mount point of instances
func (d *CiphertextCacheData) Files() FileSystem {
	return d.files
}

// DrainUploadedFilesToSafetyRaw is the drain without the goroutine at the end
func (d *CiphertextCacheData) DrainUploadedFilesToSafetyRaw() {
	if d.PermanentStorage == nil {
		d.Logger.Info("permanent storage is nil. unable to drain files to safety")
		return
	}
	//Walk through the cache, and handle .uploaded files
	fqCache := d.Files().Resolve(d.Resolve(""))
	for {
		err := Walk(
			fqCache,
			// We need to capture d because this interface won't let us pass it
			func(fqName string) (errReturn error) {
				ext := path.Ext(fqName)
				if ext == FileStateUploaded {
					d.Logger.Info("there is an uploaded file that we need to handle", zap.String("fqName", fqName))
					f, err := os.Stat(fqName)
					if err != nil {
						d.Logger.Error("there is an uploaded file that we cannot stat", zap.Error(err))
						return err
					}
					if f.IsDir() {
						d.Logger.Info("we have a directory with a .uploaded extension in the cache", zap.String("fqName", fqName))
						return nil
					}
					size := f.Size()
					fBase := path.Base(fqName)
					rName := FileId(fBase[:len(fBase)-len(ext)])
					err = d.Writeback(rName, size)
					if err != nil {
						d.Logger.Warn("error draining cache", zap.Error(err))
					}
					return err
				}
				return nil
			},
		)
		if err != nil {
			d.Logger.Warn("unable to walk cache", zap.Error(err))
		}
		time.Sleep(d.walkSleep)
	}
}

// DrainUploadedFilesToSafety moves files that were not completely sent to PermanentStorage yet, so that the instance is disposable.
// This can happen if the server reboots.
func (d *CiphertextCacheData) DrainUploadedFilesToSafety() {
	go d.DrainUploadedFilesToSafetyRaw()
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
	outFileUploaded := d.Resolve(FileName(rName + FileStateUploaded))
	key := toKey(string(d.Resolve(NewFileName(rName, ""))))

	//Get a filehandle to read the file to write back to permanent storage
	fIn, err := d.Files().Open(outFileUploaded)
	if err != nil {
		d.Logger.Warn(
			"cant writeback file",
			zap.String("filename", d.Files().Resolve(outFileUploaded)),
			zap.Error(err),
		)
		return err
	}
	defer fIn.Close()

	if size > 0 && d.PermanentStorage != nil {
		d.Logger.Debug(
			"writeback to permanent storage",
			zap.String("bucket", *d.PermanentStorage.GetName()),
			zap.String("key", *key),
		)

		err = d.PermanentStorage.Upload(fIn, key)
		if err != nil {
			d.Logger.Warn(
				"could not write to permanent storage",
				zap.String("bucket", *d.PermanentStorage.GetName()),
				zap.String("key", *key),
				zap.Error(err),
			)
			return err
		}
	}

	//Rename the file to note success
	outFileCached := d.Resolve(NewFileName(rName, FileStateCached))
	err = d.Files().Rename(outFileUploaded, outFileCached)
	if err != nil {
		d.Logger.Warn(
			"unable to rename",
			zap.String("from", d.Files().Resolve(outFileUploaded)),
			zap.String("to", d.Files().Resolve(outFileCached)),
			zap.Error(err),
		)
		return err
	}
	if d.PermanentStorage != nil {
		d.Logger.Debug(
			"permanent storage stored",
			zap.String("rname", string(rName)),
		)
	}

	return err
}

func (d *CiphertextCacheData) doDownloadFromPermanentStorage(foutCaching FileNameCached, key *string) error {
	if d.PermanentStorage == nil {
		return util.NewLoggable(PermanentStorageNotSet, nil)
	}

	//Do a whole file download from PermanentStorage
	fOut, err := d.Files().Create(foutCaching)
	defer fOut.Close()
	if err != nil {
		msg := "unable to write local buffer"
		d.Logger.Error(
			msg,
			zap.String("filename", d.Files().Resolve(foutCaching)),
			zap.Error(err),
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
	cachingPath := d.Resolve(NewFileName(rName, FileStateCaching))
	cachedPath := d.Resolve(NewFileName(rName, FileStateCached))

	logger.Info(
		"caching file",
		zap.String("filename", string(cachedPath)),
	)

	if _, err := d.Files().Stat(cachingPath); os.IsNotExist(err) {
		// Start caching the file because this is not happening already.
		err = d.Recache(rName)
		if err != nil {
			logger.Warn(
				"background recache failed",
				zap.Error(err),
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
	foutCached := d.Resolve(NewFileName(rName, FileStateCached))
	if _, err := d.Files().Stat(foutCached); os.IsExist(err) {
		return nil
	}

	// We are not supposed to be trying to get multiple copies of the same ciphertext into cache at same time
	foutCaching := d.Resolve(NewFileName(rName, FileStateCaching))
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
		if d.PermanentStorage != nil {
			if err.Error() != PermanentStorageNotFoundErrorString {
				d.Logger.Info("download from permanent storage was not successful", zap.Error(err))
			} else {
				if rName == "canary" {
					return err
				}
			}
		}
		if strings.ToLower(os.Getenv(config.OD_PEER_ENABLED)) == "true" {
			// Check p2p.... it may be there...
			var filep2p io.ReadCloser
			filep2p, err = useP2PFile(d.Logger, d.CiphertextCacheZone, rName, 0)
			if err != nil {
				d.Logger.Info("p2p cannot find", zap.Error(err))
			}
			if filep2p != nil {
				defer filep2p.Close()
				fOut, err = d.Files().Create(foutCaching)
				if err == nil {
					// We need to copy the *whole* file in this case.
					_, err = io.Copy(fOut, filep2p)
					fOut.Close()
					if err != nil {
						d.Logger.Info("p2p recache failed", zap.Error(err))
					} else {
						d.Logger.Info("p2p recache success")
					}
					// leave err where it is.
				}
			} else {
				if d.PermanentStorage == nil {
					// single node without permanent storage and does not have
					return nil
				}
			}
		}
	}

	// This only exists for exotic corner cases.  Without network errors,
	// this block should be unreachable.
	tries := 4 // 4=~15 seconds max; 8 = ~ 2mins max
	waitTime := 1 * time.Second
	prevWaitTime := 0 * time.Second
	for tries > 0 && err != nil && d.PermanentStorage != nil {
		err = d.doDownloadFromPermanentStorage(foutCaching, key)
		tries--
		if err == nil {
			break
		} else {
			d.Logger.Error(
				"unable to download out of permanent storage or p2p",
				zap.Duration("seconds", waitTime/(time.Second)),
				zap.Int("more tries", tries-1),
				zap.String("key", string(rName)),
				zap.Error(err),
			)
			// Without a file length, this is our best guess
			time.Sleep(waitTime)
			// Fibonacci progression 1 1 2 3 ... ... 8 of them gives a total wait time of about 2 mins
			oldWaitTime := waitTime
			waitTime = prevWaitTime + waitTime
			prevWaitTime = oldWaitTime
		}
	}

	if err != nil {
		d.Logger.Error("giving up on recaching file", zap.Error(err))
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
func CacheMustExist(d CiphertextCache, logger *zap.Logger) (err error) {
	if _, err = d.Files().Stat(d.Resolve("")); os.IsNotExist(err) {
		err = d.Files().MkdirAll(d.Resolve(""), os.FileMode(int(0700)))
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

// CacheInventory writes an inventory of what's in the cache to a writer for the stats page
func (d *CiphertextCacheData) CacheInventory(w io.Writer, verbose bool) {
	fqCache := d.Files().Resolve(d.Resolve(""))
	fmt.Fprintf(w, "\n\ncache at %s on %s\n", fqCache, config.NodeID)
	Walk(
		fqCache,
		func(fqName string) error {
			if strings.Compare(fqName, fqCache) != 0 {
				if verbose || strings.HasSuffix(fqName, FileStateUploaded) {
					fmt.Fprintf(w, "%s\n", fqName)
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
	Walk(
		fqCache,
		func(name string) error {
			if strings.Compare(name, fqCache) != 0 {
				if strings.HasSuffix(name, FileStateUploaded) {
					fi, e := os.Stat(name)
					if e != nil {
						return e
					}
					if fi.Size() > 0 {
						uploaded++
					}
				}
			}
			return nil
		},
	)
	return uploaded
}

// GetPermanentStorage gets the permanent storage provider plugged into this cache
func (d *CiphertextCacheData) GetPermanentStorage() PermanentStorage {
	return d.PermanentStorage
}

// GetCiphertextCacheZone is the key that this is stored under
func (d *CiphertextCacheData) GetCiphertextCacheZone() CiphertextCacheZone {
	return d.CiphertextCacheZone
}

// SetCiphertextCacheZone sets the key by which we actually do the lookup
func (d *CiphertextCacheData) SetCiphertextCacheZone(zone CiphertextCacheZone) {
	d.CiphertextCacheZone = zone
}

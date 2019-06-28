package ciphertext

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"syscall"

	"go.uber.org/zap"
)

type cachePurgeStats struct {
	started       time.Time
	deletedCount  int64
	deletedSize   int64
	errorCount    int64
	errorSize     int64
	reviewedCount int64
	reviewedSize  int64
}

const oneWeek = int64(60 * 60 * 24 * 7)

// CachePurge will periodically delete files that do not need to be in the cache.
func (d *CiphertextCacheData) CachePurge() {
	if d.PermanentStorage == nil {
		d.Logger.Info("cachepurge found that permanent storage is nil.  purge is disabled.")
		return
	}
	d.Logger.Info("cachepurge start")
	// read from environment variables:
	//    lowThresholdPercent (floating point 0..1)
	//    highThresholdPercent (floating point 0..1)
	//    ageEligibleForEviction (integer seconds)
	for {
		d.Logger.Debug("cachepurge iteration begin")
		//Get the current disk space usage
		fqCache := d.Files().Resolve(d.Resolve(""))
		d.Logger.Debug("cachepurge calling syscall.statfs")
		sfs := syscall.Statfs_t{}
		err := syscall.Statfs(fqCache, &sfs)
		d.Logger.Debug("cachepurge got statfs data")
		if err != nil {
			d.Logger.Error(
				"cachepurge unable to purge on statfs fail",
				zap.String("filename", fqCache),
				zap.Error(err),
			)
		} else {
			//Fraction of disk used
			usage := 1.0 - float64(sfs.Bavail)/float64(sfs.Blocks)

			availabledisksize := (sfs.Bsize * int64(sfs.Bavail))
			totaldisksize := (sfs.Bsize * int64(sfs.Blocks))
			useddisksize := totaldisksize - availabledisksize
			//useddiskpercent := float64(float64(useddisksize) / float64(totaldisksize))

			d.Logger.Debug("cachepurge iteration check",
				zap.String("fqCache", fqCache),
				zap.Float64("usage", usage),
				zap.Float64("thresholdLow", d.lowThresholdPercent),
				zap.Float64("thresholdHigh", d.highThresholdPercent),
				zap.Int64("diskBytesUsed", useddisksize),
				zap.Int64("diskBytesAvailable", availabledisksize),
				zap.Int64("diskBytesTotal", totaldisksize))

			//don't even bother walking if we are below the low ThresholdPercent and dont have a file limit
			if usage > d.lowThresholdPercent || d.fileLimit > 0 {
				cachePurgeIteration(d, usage)
			} else {
				d.Logger.Debug("cachepurge has no work to do, below low ThresholdPercent and have no file limit")
			}
		}
		time.Sleep(d.walkSleep)
	}
}

func cachePurgeIteration(d *CiphertextCacheData, usage float64) {
	fqCache := d.Files().Resolve(d.Resolve(""))
	cpsTotal := cachePurgeStats{started: time.Now().UTC()}
	err := Walk(
		fqCache,
		func(fqName string) (errReturn error) {
			err := filePurgeVisit(d, fqName, usage, &cpsTotal)
			return err
		},
	)
	cachePurgeDuration := time.Since(cpsTotal.started)
	d.Logger.Debug("cachepurge iteration done",
		zap.String("fqCache", fqCache),
		zap.Duration("duration", cachePurgeDuration),
		zap.Int64("deletedCount", cpsTotal.deletedCount),
		zap.Int64("deletedSize", cpsTotal.deletedSize),
		zap.Int64("errorCount", cpsTotal.errorCount),
		zap.Int64("errorSize", cpsTotal.errorSize),
		zap.Int64("reviewedCount", cpsTotal.reviewedCount),
		zap.Int64("reviewedSize", cpsTotal.reviewedSize))
	if err != nil {
		d.Logger.Error(
			"cachepurge unable to walk cache",
			zap.String("filename", fqCache),
			zap.Error(err),
		)
	}
}

type Walker func(fqName string) error

func Walk(fqCache string, walker Walker) error {
	d, err := os.Open(fqCache)
	// DIMEODS-1262 - Ensure file closed if not nil
	if d != nil {
		defer d.Close()
	}
	if err != nil {
		return err
	}
	for {
		namesInDirectory, err := d.Readdirnames(400)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		for _, name := range namesInDirectory {
			walker(fmt.Sprintf("%s/%s", fqCache, name))
		}
	}
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
//  disk usage below lowThresholdPercent: ignore this file unless there's an overall file limit
//  disk usage above highThresholdPercent: delete this file if it's old enough for eviction
//  disk usage in range of ThresholdPercents:
//      if file is too fresh to evict at all: ignore this file
//      if int(size/(age*age)) == 0: delete this file
//
//  The net effect is that some files remain fresh because they were recently uploaded or fetched.
//  Multiplying times filesize recognizes that the penalty for deleting a large file is proportional
//  to its size.  But because 1/(age*age) drops rapidly, even very large files will quickly become
//  eligible for deletion if not used.  If there are many files that are accessed often, then the large files
//  will be selected for deletion.  Large files that keep getting used will stay in cache
//  as long as they keep getting used.  Because they take N times longer to get stamped due to
//  the length of the file transfer, it is fair to make it take N times longer to get evicted.
//
// The following is a simplified table of how long a file may take to be evicted based on
// its size if it isn't accessed and the cache needs to make room to stay within thresholds
//		20 KB				2.5 minutes
//		400 KB				11 minutes
//		1 MB				17 minutes
//		5 MB				37.5 minutes
//		20 MB				1 hour, 15 minutes
//		100 MB				2 hours, 47 minutes
//		1 GB				About 9 hours
//		4 GB				About 18 hours
//
//  The graph will look like a sawtooth between lowThresholdPercent and highThresholdPercent, where there is
//  a delay in size drops that is dependent on size and doubly dependent on age since last access.
//  Size and Age prioritize what is still sitting in cache when we hit lowThresholdPercent.
//
func filePurgeVisit(d *CiphertextCacheData, fqName string, usage float64, cps *cachePurgeStats) (errReturn error) {

	// Apply file sleep time before performing this check
	time.Sleep(d.fileSleep)

	cps.reviewedCount++
	f, err := os.Stat(fqName)
	if err != nil {
		if strings.HasSuffix(err.Error(), "no such file or directory") {
			// This can happen in the background from other routines
			// 		uploading -> uploaded
			// 		uploaded -> cached
			// 		caching -> cached
			// 		uploaded -> orphaned
			// We can ignore this and allow a subsequent cachepurge iteration to check
			return nil
		}
		// Log this unhandled error and return
		d.Logger.Error(
			"cachepurge unable to stat file",
			zap.String("filename", fqName),
			zap.Error(err),
		)
		cps.errorCount++
		return nil
	}

	//Ignore directories.  We should not have an unbounded number of directories.
	//And we must ignore h.CacheLocation
	if f.IsDir() || !f.Mode().IsRegular() {
		return nil
	}

	//Size and time since last accessed determine the value of the file
	t := f.ModTime().UTC().Unix() //In units of second
	n := time.Now().UTC().Unix()  //In units of second
	ageInSeconds := (n - t) + 1   // Ensure > 0
	size := f.Size()
	cps.reviewedSize += size

	// Action based on file extension which denotes state in the cache
	ext := path.Ext(string(fqName))
	switch {
	//Note that cached files are persistently stored already
	case ext == FileStateCached:
		// Remove if above high threshold percent, or if aged and above the low threshold percent
		// Limit for file upload size is effectively the space between high threshold percent and disk filled.
		oldEnoughToEvict := (ageInSeconds > d.ageEligibleForEviction)
		hitFileLimit := (d.fileLimit > 0 && cps.reviewedCount > d.fileLimit)
		fullEnoughToEvict := (usage > d.lowThresholdPercent || hitFileLimit)
		if oldEnoughToEvict && fullEnoughToEvict {
			// Determine if should evict by age compared to file size
			sizeAgeEviction := (size / (ageInSeconds * ageInSeconds)) < 1
			// Random selection when usage above high treshold percent
			// ie: highThresholdPercent 0.9, lowThresholdPercent 0.7, usage 0.95:
			//   usage is 0.05 over high threshold percent,
			//   usage is 0.25 over low threshold percent
			//   so delete a percentage of the data randomly by low threshold based upon usage over the high threshold
			randomlySelected := ((usage-d.highThresholdPercent) > 0 && rand.Float64() < (usage-d.lowThresholdPercent))
			// Determine if should evict by disk threshold limits
			diskLimitEviction := randomlySelected || hitFileLimit

			if sizeAgeEviction || diskLimitEviction {
				//Name is fully qualified, so use os call!
				errReturn := os.Remove(fqName)
				if errReturn != nil {
					cps.errorCount++
					cps.errorSize += size
					d.Logger.Error(
						"cachepurge unable to purge cached file",
						zap.String("filename", fqName),
						zap.Error(errReturn),
					)
					attemptToEmptyFile(d, fqName)
					return nil
				}
				cps.deletedCount++
				cps.deletedSize += size
				d.Logger.Info(
					"cachepurge removed file",
					zap.String("filename", fqName),
					zap.Int64("ageinseconds", ageInSeconds),
					zap.Int64("size", size),
					zap.Float64("usage", usage),
				)
			}
		}
	case ext == FileStateOrphaned:
		if _, err := os.Stat(fqName); err == nil {
			errReturn := os.Remove(fqName)
			if errReturn != nil {
				cps.errorCount++
				cps.errorSize += size
				d.Logger.Error("cachepurge unable to purge orphaned file", zap.String("filename", fqName), zap.Error(errReturn))
				attemptToEmptyFile(d, fqName)
				return nil
			}
			cps.deletedCount++
			cps.deletedSize += size
		}
	case ext == FileStateUploaded:
		if ageInSeconds > oneWeek {
			cps.errorCount++
			cps.errorSize += size
			// There is something clearly wrong here.  Log it
			d.Logger.Error("ciphertextcache file not uploaded after a week", zap.String("filename", fqName))
			return nil
		}
	default:
		//If something has been here for a week, and it's not cached, then it's
		//garbage.  If a machine has been turned off for a few days, the files
		//might legitimately be awaiting upload.  Other states are certainly
		//garbage after only a few hours.
		if ageInSeconds > oneWeek {
			if _, err := os.Stat(fqName); err == nil {
				errReturn := os.Remove(fqName)
				if errReturn != nil {
					cps.errorCount++
					cps.errorSize += size
					d.Logger.Error(
						"cachepurge unable to purge",
						zap.String("filename", fqName),
						zap.Error(errReturn),
					)
					attemptToEmptyFile(d, fqName)
					return nil
				} else {
					cps.deletedCount++
					cps.deletedSize += size
				}
				//Count this anomaly
				d.Logger.Warn(
					"cachepurge removed file for age > oneweek",
					zap.String("filename", fqName),
					zap.Int64("age", ageInSeconds),
					zap.Float64("usage", usage),
				)
			}
			return nil
		}
	}
	return nil
}

func attemptToEmptyFile(d *CiphertextCacheData, fqName string) {
	// Sanity check to make sure this isn't being called for a file in the uploaded state.
	// As this function will truncate its contents, we don't want to risk having a valid upload
	// have its file contents wiped out before evacuated to PermanentStorage
	ext := path.Ext(string(fqName))
	if ext == FileStateUploaded {
		d.Logger.Warn("Logic Error! attemptToEmptyFile was called on a file in Uploaded state! Skipping")
		return
	}
	if _, err := os.Stat(fqName); err == nil {
		e := os.Truncate(fqName, 0)
		if e != nil {
			d.Logger.Error("unable to empty file", zap.String("filename", fqName), zap.Error(e))
		}
		d.Logger.Info("truncated file to free space", zap.String("filename", fqName))
	}
}

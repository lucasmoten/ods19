package server

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/config"
	oduconfig "decipher.com/object-drive-server/config"

	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	// PurgeAnomaly error code given when we purged something that wasn't cleaned up
	PurgeAnomaly = 1500
	// FailPurgeAnomaly error code given when we failed to purge something that wasn't cleaned up
	FailPurgeAnomaly = 1501
	// FailCacheWalk error code given when we tried to walk cache, and something went wrong
	FailCacheWalk = 1502
	// FailDrainToCache error code given when we could not drain to cache
	FailDrainToCache = 1503
	// FailCacheToDrain error code given when we could not cache to drain
	FailCacheToDrain = 1504
	// FailS3Download error code given when we failed to download out of S3
	FailS3Download = 1505
)

// checkAWSEnvironmentVars prevents the server from starting if appropriate vars
// are not set.
func checkAWSEnvironmentVars() {
	// Variables for the environment can be provided as either the native AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
	// or be prefixed with the common "OD_" as in OD_AWS_REGION, OD_AWS_ACCESS_KEY_ID, and OD_AWS_SECRET_ACCESS_KEY
	// Environment variables will be normalized to the AWS_ variants to facilitate internal library calls
	region := oduconfig.GetEnvOrDefault("OD_AWS_REGION", oduconfig.GetEnvOrDefault("AWS_REGION", ""))
	if len(region) > 0 {
		os.Setenv("AWS_REGION", region)
	}
	accessKeyID := oduconfig.GetEnvOrDefault("OD_AWS_ACCESS_KEY_ID", oduconfig.GetEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	if len(accessKeyID) > 0 {
		os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	}
	secretKey := oduconfig.GetEnvOrDefault("OD_AWS_SECRET_ACCESS_KEY", oduconfig.GetEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""))
	if len(secretKey) > 0 {
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	}
	// If any values are not set, then this is a fatal error
	if region == "" || accessKeyID == "" || secretKey == "" {
		log.Fatal("Fatal Error: Environment variables AWS_REGION, AWS_SECRET_ACCESS_KEY, and AWS_ACCESS_KEY_ID must be set.")
	}
	return
}

// NullDrainProviderData is just a file location that does not talk to S3.
type NullDrainProviderData struct {
	//Where the CacheLocation is rooted on disk (ie: a very large drive mounted)
	CacheObject DrainCache
	//Location of the cache
	CacheLocationString string
}

// S3DrainProviderData moves data from cache to the drain... S3 buckets in this case.
type S3DrainProviderData struct {
	//Where the CacheLocation is rooted on disk (ie: a very large drive mounted)
	CacheObject DrainCache
	//Location of the cache
	CacheLocationString string
	//The connection to S3
	AWSSession *session.Session

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
}

// NewS3DrainProvider sets up a drain with default parameters overridden by environment variables
// TODO this should return an error, as well.
func NewS3DrainProvider(root, name string) DrainProvider {
	var err error
	lowWatermark := 0.50
	lowWatermarkSuggested := oduconfig.GetEnvOrDefault("OD_CACHE_LOWWATERMARK", "0.50")
	if len(lowWatermarkSuggested) > 0 {
		lowWatermark, err = strconv.ParseFloat(lowWatermarkSuggested, 32)
		if err != nil {
			log.Printf("!! Unable to set lowWatermark to %s:%v", lowWatermarkSuggested, err)
		}
	}
	highWatermark := 0.75
	highWatermarkSuggested := oduconfig.GetEnvOrDefault("OD_CACHE_HIGHWATERMARK", "0.75")
	if len(highWatermarkSuggested) > 0 {
		highWatermark, err = strconv.ParseFloat(highWatermarkSuggested, 32)
		if err != nil {
			log.Printf("!! Unable to set highWatermark to %s:%v", highWatermarkSuggested, err)
		}
	}
	ageEligibleForEviction := int64(60 * 5)
	ageEligibleForEvictionSuggested := oduconfig.GetEnvOrDefault("OD_CACHE_EVICTAGE", "300")
	if len(ageEligibleForEvictionSuggested) > 0 {
		ageEligibleForEviction, err = strconv.ParseInt(ageEligibleForEvictionSuggested, 10, 64)
		if err != nil {
			log.Printf("!! Unable to set highWatermark to %s:%v", ageEligibleForEvictionSuggested, err)
		}
	}
	walkSleep := time.Duration(30 * time.Second)
	walkSleepSuggested := oduconfig.GetEnvOrDefault("OD_CACHE_WALKSLEEP", "30")
	if len(walkSleepSuggested) > 0 {
		walkSleepInt, err := strconv.ParseInt(walkSleepSuggested, 10, 64)
		if err != nil {
			log.Printf("!! Unable to set walkSleep to %d:%v", walkSleepInt, err)
		}
		walkSleep = time.Duration(time.Duration(walkSleepInt) * time.Second)
	}
	d := NewS3DrainProviderRaw(root, name, lowWatermark, ageEligibleForEviction, highWatermark, walkSleep)
	go d.DrainUploadedFilesToSafety()
	return d
}

// NewS3DrainProviderRaw set up a new drain provider that gives us members to use the drain and goroutine to clean cache.
// Call this to build a test cache.
func NewS3DrainProviderRaw(root, name string, lowWatermark float64, ageEligibleForEviction int64, highWatermark float64, walkSleep time.Duration) *S3DrainProviderData {
	checkAWSEnvironmentVars()

	d := &S3DrainProviderData{
		AWSSession:             awsS3(),
		CacheObject:            DrainCacheData{root},
		CacheLocationString:    name,
		lowWatermark:           lowWatermark,
		ageEligibleForEviction: ageEligibleForEviction,
		highWatermark:          highWatermark,
		walkSleep:              walkSleep,
	}
	CacheMustExist(d)
	log.Printf(
		"starting CachePurge: lowWatermark:%f ofDisk, highWatermark:%f ofDisk, ageEligibleForEviction:%d sec walkSleep:%d",
		lowWatermark,
		highWatermark,
		ageEligibleForEviction,
		walkSleep,
	)
	return d
}

// Resolve a name to somewhere in the cache, given the rName
func (d *S3DrainProviderData) Resolve(fName FileName) FileNameCached {
	return FileNameCached(d.CacheLocationString + "/" + string(fName))
}

// Resolve a name to somewhere in the cache, given the rName
func (d *NullDrainProviderData) Resolve(fName FileName) FileNameCached {
	return FileNameCached(d.CacheLocationString + "/" + string(fName))
}

// awsS3 just gets us a session.
//This is account as in the ["default"] entry in ~/.aws/credentials
func awsS3() *session.Session {
	sessionConfig := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
	}
	return session.New(sessionConfig)
}

// NewNullDrainProvider setup a drain provider that doesnt use S3 backend, just local caching.
func NewNullDrainProvider(root, name string) DrainProvider {
	d := &NullDrainProviderData{
		CacheObject:         DrainCacheData{root},
		CacheLocationString: name,
	}
	CacheMustExist(d)
	//there is no goroutine to purge, because there was no place to drain off to
	return d
}

// Files is the mount point of instances
func (d *S3DrainProviderData) Files() DrainCache {
	return d.CacheObject
}

// Files is the mount point of cache instances
func (d *NullDrainProviderData) Files() DrainCache {
	return d.CacheObject
}

// CacheToDrain just renames the file without moving it for the null implementation.
func (d *NullDrainProviderData) CacheToDrain(
	bucket *string,
	rName FileId,
	size int64,
) error {
	var err error
	outFileUploaded := d.Resolve(NewFileName(rName, ".uploaded"))
	outFileCached := d.Resolve(NewFileName(rName, ".cached"))
	err = d.Files().Rename(outFileUploaded, outFileCached)
	if err != nil {
		log.Printf("Unable to rename uploaded file %s %v:", d.Files().Resolve(outFileUploaded), err)
		return err
	}
	return nil
}

// DrainUploadedFilesToSafety moves files that were not completely sent to S3 into S3.
// This can happen if the server reboots.
func (d *S3DrainProviderData) DrainUploadedFilesToSafety() {
	//Walk through the cache, and handle .uploaded files
	fqCache := d.Files().Resolve(d.Resolve(""))
	err := filepath.Walk(
		fqCache,
		// We need to capture d because this interface won't let us pass it
		func(fqName string, f os.FileInfo, err error) (errReturn error) {
			if err != nil {
				msg := fmt.Sprintf("Error walking directory on initial upload for %s", fqName)
				sendAppErrorResponse(nil, NewAppError(FailCacheWalk, err, msg))
				// I didn't generate this error, so I am assuming that I can just log the problem.
				// TODO: this error is not being counted
				return nil
			}

			if f.IsDir() {
				return nil
			}
			size := f.Size()
			ext := path.Ext(fqName)
			bucket := &config.DefaultBucket
			if ext == ".uploaded" {
				err := d.CacheToDrain(bucket, FileId(path.Base(fqName)), size)
				if err != nil {
					msg := fmt.Sprintf("error draining cache")
					sendAppErrorResponse(nil, NewAppError(FailCacheToDrain, err, msg))
				}
			}
			if ext == ".caching" || ext == ".uploading" {
				//On startup, any .caching files are associated with now-dead goroutines.
				//On startup, any .uploading files are associated with now-dead uploads.
				os.Remove(fqName)
			}
			return nil
		},
	)
	if err != nil {
		sendAppErrorResponse(nil, NewAppError(FailCacheWalk, err, "Unable to walk cache"))
	}
	//Only now can we start to purge files
	go d.CachePurge()
}

// CacheToDrain drains to S3.  Note that because this is async with respect to the http session,
// we cant return AppError.
//
// Dont delete the file here if something goes wrong... because the caller tries this multiple times
//
func (d *S3DrainProviderData) CacheToDrain(
	bucket *string,
	rName FileId,
	size int64,
) error {
	sess := d.AWSSession
	outFileUploaded := d.Resolve(FileName(rName + ".uploaded"))

	fIn, err := d.Files().Open(outFileUploaded)
	if err != nil {
		log.Printf("Cant drain off file: %v", err)
		return err
	}
	defer fIn.Close()

	key := aws.String(string(d.Resolve(NewFileName(rName, ""))))
	log.Printf("draining to S3 %s: %s", *bucket, *key)

	uploader := s3manager.NewUploader(sess)
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   fIn,
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		log.Printf("Could not write to S3: %v", err)
		return err
	}

	//Rename the file to note that it only lives here as cached for download
	//It might be deleted at any time
	outFileCached := d.Resolve(NewFileName(rName, ".cached"))
	err = d.Files().Rename(outFileUploaded, outFileCached)
	if err != nil {
		log.Printf("Unable to rename uploaded file %s %v:", outFileUploaded, err)
		return err
	}
	log.Printf("rename:%s -> %s", outFileUploaded, outFileCached)

	log.Printf("Uploaded to %v: %v", *bucket, result.Location)
	return err
}

// DrainToCache does nothing for a null drain which leaves files local.
func (d *NullDrainProviderData) DrainToCache(
	bucket *string,
	theFile FileId,
) (*AppError, error) {
	return nil, nil
}

/*
// CacheLocation is where the local cache lives. (S3 within bucket or filesystem path)
func (d *NullDrainProviderData) Cache() string {
	return d.CacheLocationString
}
*/

//TODO: without a file length to expect, we are only making guesses as to how long we can wait.
func S3DownloadAttempt(downloader *s3manager.Downloader, fOut *os.File, bucket *string, key *string) (int64, error) {
	length, err := downloader.Download(
		fOut,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		},
	)
	if err != nil {
		log.Printf("Unable to download out of S3 bucket %v: %s", *bucket, *key)
	}
	return length, err
}

// DrainToCache gets a file back out of the drain into the cache.
func (d *S3DrainProviderData) DrainToCache(
	bucket *string,
	rName FileId,
) (*AppError, error) {
	log.Printf("Get from S3 bucket %s: %s", *bucket, rName)
	// This file must ONLY exist for the duration of this function.
	// we must remove it or rename it before we exit.
	foutCaching := d.Resolve(NewFileName(rName, ".caching"))
	foutCached := d.Resolve(NewFileName(rName, ".cached"))
	fOut, err := d.Files().Create(foutCaching)
	if err != nil {
		msg := fmt.Sprintf("Unable to write local buffer file %s", foutCaching)
		sendAppErrorResponse(nil, NewAppError(FailDrainToCache, err, msg))
		return nil, err
	}
	defer d.Files().Remove(foutCaching)
	defer fOut.Close()

	//Try to download it a few times, doubling our willingness to wait each time
	//files move up into S3 at 30MB/s
	tries := 22
	waitTime := 1 * time.Second
	prevWaitTime := 0 * time.Second
	key := aws.String(string(d.Resolve(NewFileName(rName, ""))))
	downloader := s3manager.NewDownloader(d.AWSSession)
	for tries > 0 {
		_, err = S3DownloadAttempt(downloader, fOut, bucket, key)
		tries--
		if err == nil {
			break
		} else {
			log.Printf("Unable to download out of S3 bucket %v. Trying again in %ds, %d more attempts possible: %s", *bucket, waitTime/(time.Second), tries-1, rName)
			//Without a file length, this is our best guess
			time.Sleep(waitTime)
			//Fibonacci progression 1 1 2 3 ... ... 22 of them gives a total wait time of about 2 mins, or almost 8GB
			oldWaitTime := waitTime
			waitTime = prevWaitTime + waitTime
			prevWaitTime = oldWaitTime
		}
	}
	if err != nil {
		return NewAppError(500, err, fmt.Sprintf("Give up to get %s out of cache", rName)), err
	}

	//Signal that we finally cached the file
	err = d.Files().Rename(foutCaching, foutCached)
	if err != nil {
		log.Printf("Failed to rename from %s to %s", foutCaching, foutCached)
	}
	log.Printf("rename:%s -> %s", foutCaching, foutCached)
	return nil, nil
}

/*
// CacheLocation gives the file location locally, and in the buckets
func (d *S3DrainProviderData) Cache() string {
	return d.CacheLocationString
}
*/

// CacheMustExist ensures that the cache directory exists.
func CacheMustExist(d DrainProvider) (err error) {
	if _, err = d.Files().Stat(d.Resolve("")); os.IsNotExist(err) {
		err = d.Files().Mkdir(d.Resolve(""), 0700)
		log.Printf("Creating cache directory %s", d.Files().Resolve(d.Resolve("")))
		if err != nil {
			log.Printf("Cannot create cache directory: %v", err)
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
func filePurgeVisit(d *S3DrainProviderData, fqName string, f os.FileInfo, err error) (errReturn error) {
	if err != nil {
		log.Printf("Error walking directory for %s: %v", fqName, err)
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
		msg := fmt.Sprintf("Unable to purge %s, due to statfs fail", fqName)
		sendAppErrorResponse(nil, NewAppError(FailPurgeAnomaly, nil, msg))
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
					msg := fmt.Sprintf("Unable to purge %s", fqName)
					sendAppErrorResponse(nil, NewAppError(FailPurgeAnomaly, errReturn, msg))
					return nil
				} else {
					log.Printf("Purged %s.  Age:%ds Size:%d DiskUsage:%f", fqName, ageInSeconds, size, usage)
				}
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
				msg := fmt.Sprintf("Unable to purge %s", fqName)
				sendAppErrorResponse(nil, NewAppError(FailPurgeAnomaly, errReturn, msg))
				return nil
			} else {
				msg := fmt.Sprintf("Purged for age %s.  Age:%ds Size:%d DiskUsage:%f", fqName, ageInSeconds, size, usage)
				//Count this anomaly
				sendAppErrorResponse(nil, NewAppError(PurgeAnomaly, nil, msg))
				return nil
			}
		}
	}
	return
}

// CachePurge will periodically delete files that do not need to be in the cache.
func (d *S3DrainProviderData) CachePurge() {
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
			log.Printf("Unable to walk cache %s: %v", fqCache, err)
		}
		time.Sleep(d.walkSleep)
	}
}

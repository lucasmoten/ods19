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

	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	//We purged something that wasn't cleaned up
	PurgeAnomaly = 1500
	//We failed to purge something that wasn't cleaned up
	FailPurgeAnomaly = 1501
	//We tried to walk cache, and something went wrong
	FailCacheWalk = 1502
	//We could not drain to cache
	FailDrainToCache = 1503
	//We could not cache to drain
	FailCacheToDrain = 1504
	//Failed to download out of S3
	FailS3Download = 1505
)

// checkAWSEnvironmentVars prevents the server from starting if appropriate vars
// are not set.
func checkAWSEnvironmentVars() {
	region := os.Getenv("AWS_REGION")
	secretKey := os.Getenv("AWS_SECRET_KEY")
	secretKeyAlt := os.Getenv("AWS_SECRET_ACCESS_KEY")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	if region == "" || (secretKey == "" && secretKeyAlt == "") || accessKeyID == "" {
		log.Fatal("Fatal Error: Environment variables AWS_REGION, AWS_SECRET_KEY, and AWS_ACCESS_KEY_ID must be set.")
	}
	return
}

// NullDrainProviderData is just a file location that does not talk to S3.
type NullDrainProviderData struct {
	//Location of the cache
	CacheLocationString string
}

// S3DrainProviderData moves data from cache to the drain... S3 buckets in this case.
type S3DrainProviderData struct {
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
func NewS3DrainProvider(name string) DrainProvider {
	var err error
	lowWatermark := 0.50
	lowWatermarkSuggested := os.Getenv("lowWatermark")
	if len(lowWatermarkSuggested) > 0 {
		lowWatermark, err = strconv.ParseFloat(lowWatermarkSuggested, 32)
		if err != nil {
			log.Printf("!! Unable to set lowWatermark to %s:%v", lowWatermarkSuggested, err)
		}
	}
	highWatermark := 0.75
	highWatermarkSuggested := os.Getenv("highWatermark")
	if len(highWatermarkSuggested) > 0 {
		highWatermark, err = strconv.ParseFloat(highWatermarkSuggested, 32)
		if err != nil {
			log.Printf("!! Unable to set highWatermark to %s:%v", highWatermarkSuggested, err)
		}
	}
	ageEligibleForEviction := int64(60 * 5)
	ageEligibleForEvictionSuggested := os.Getenv("ageEligibleForEviction")
	if len(ageEligibleForEvictionSuggested) > 0 {
		ageEligibleForEviction, err = strconv.ParseInt(ageEligibleForEvictionSuggested, 10, 64)
		if err != nil {
			log.Printf("!! Unable to set highWatermark to %s:%v", ageEligibleForEvictionSuggested, err)
		}
	}
	walkSleep := time.Duration(30 * time.Second)
	walkSleepSuggested := os.Getenv("walkSleep")
	if len(walkSleepSuggested) > 0 {
		walkSleepInt, err := strconv.ParseInt(walkSleepSuggested, 10, 64)
		if err != nil {
			log.Printf("!! Unable to set walkSleep to %s:%v", walkSleepInt, err)
		}
		walkSleep = time.Duration(time.Duration(walkSleepInt) * time.Second)
	}
	d := NewS3DrainProviderRaw(name, lowWatermark, ageEligibleForEviction, highWatermark, walkSleep)
	go d.DrainUploadedFilesToSafety()
	return d
}

// NewS3DrainProviderRaw set up a new drain provider that gives us members to use the drain and goroutine to clean cache.
// Call this to build a test cache.
func NewS3DrainProviderRaw(name string, lowWatermark float64, ageEligibleForEviction int64, highWatermark float64, walkSleep time.Duration) *S3DrainProviderData {
	checkAWSEnvironmentVars()

	d := &S3DrainProviderData{
		AWSSession:             awsS3(),
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

// awsS3 just gets us a session.
//This is account as in the ["default"] entry in ~/.aws/credentials
func awsS3() *session.Session {
	sessionConfig := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
	}
	return session.New(sessionConfig)
}

// NewNullDrainProvider setup a drain provider that doesnt use S3 backend, just local caching.
func NewNullDrainProvider(name string) DrainProvider {
	d := &NullDrainProviderData{
		CacheLocationString: name,
	}
	CacheMustExist(d)
	//there is no goroutine to purge, because there was no place to drain off to
	return d
}

// CacheToDrain just renames the file without moving it for the null implementation.
func (d *NullDrainProviderData) CacheToDrain(
	bucket *string,
	rName string,
	size int64,
) error {
	var err error
	outFileUploaded := d.CacheLocationString + "/" + rName + ".uploaded"
	outFileCached := d.CacheLocationString + "/" + rName + ".cached"
	err = os.Rename(outFileUploaded, outFileCached)
	if err != nil {
		log.Printf("Unable to rename uploaded file %s %v:", outFileUploaded, err)
		return err
	}
	return nil
}

// DrainUploadedFilesToSafety moves files that were not completely sent to S3 into S3.
// This can happen if the server reboots.
func (d *S3DrainProviderData) DrainUploadedFilesToSafety() {
	//Walk through the cache, and handle .uploaded files
	err := filepath.Walk(
		d.CacheLocationString,
		// We need to capture d because this interface won't let us pass it
		func(name string, f os.FileInfo, err error) (errReturn error) {
			if err != nil {
				msg := fmt.Sprintf("Error walking directory on initial upload for %s: %v", name, err)
				sendAppErrorResponse(nil, NewAppError(FailCacheWalk, err, msg))
				// I didn't generate this error, so I am assuming that I can just log the problem.
				// TODO: this error is not being counted
				return nil
			}

			if f.IsDir() {
				return nil
			}
			size := f.Size()
			ext := path.Ext(name)
			bucket := &config.DefaultBucket
			if ext == ".uploaded" {
				err := d.CacheToDrain(bucket, name, size)
				if err != nil {
					msg := fmt.Sprintf("error draining cache:%v", err)
					sendAppErrorResponse(nil, NewAppError(FailCacheToDrain, err, msg))
				}
			}
			if ext == ".caching" || ext == ".uploading" {
				//On startup, any .caching files are associated with now-dead goroutines.
				//On startup, any .uploading files are associated with now-dead uploads.
				os.Remove(name)
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
func (d *S3DrainProviderData) CacheToDrain(
	bucket *string,
	rName string,
	size int64,
) error {
	sess := d.AWSSession
	outFileUploaded := d.CacheLocationString + "/" + rName + ".uploaded"

	fIn, err := os.Open(outFileUploaded)
	if err != nil {
		log.Printf("Cant drain off file: %v", err)
		return err
	}
	defer fIn.Close()
	log.Printf("draining to S3 %s: %s", *bucket, rName)

	uploader := s3manager.NewUploader(sess)
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   fIn,
		Bucket: bucket,
		Key:    aws.String(d.CacheLocationString + "/" + rName),
	})
	if err != nil {
		log.Printf("Could not write to S3: %v", err)
		return err
	}

	//Rename the file to note that it only lives here as cached for download
	//It might be deleted at any time
	outFileCached := d.CacheLocationString + "/" + rName + ".cached"
	err = os.Rename(outFileUploaded, outFileCached)
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
	theFile string,
) (*AppError, error) {
	return nil, nil
}

// CacheLocation is where the local cache lives.
func (d *NullDrainProviderData) CacheLocation() string {
	return d.CacheLocationString
}

// DrainToCache gets a file back out of the drain into the cache.
func (d *S3DrainProviderData) DrainToCache(
	bucket *string,
	theFile string,
) (*AppError, error) {
	log.Printf("Get from S3 bucket %s: %s", *bucket, theFile)
	// This file must ONLY exist for the duration of this function.
	// we must remove it or rename it before we exit.
	foutCaching := d.CacheLocationString + "/" + theFile + ".caching"
	foutCached := d.CacheLocationString + "/" + theFile + ".cached"
	fOut, err := os.Create(foutCaching)
	if err != nil {
		msg := fmt.Sprintf("Unable to write local buffer file %s: %v", theFile, err)
		sendAppErrorResponse(nil, NewAppError(FailDrainToCache, err, msg))
	}
	defer fOut.Close()

	downloader := s3manager.NewDownloader(d.AWSSession)
	_, err = downloader.Download(
		fOut,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    aws.String(d.CacheLocationString + "/" + theFile),
		},
	)
	if err != nil {
		log.Printf("Unable to download out of S3 bucket %v: %v", *bucket, theFile)
		//Do not signal that a goroutine is still working on caching this file
		os.Remove(foutCaching)
		return NewAppError(500, err, fmt.Sprintf("Unable to get %s out of cache", theFile)), err
	}
	//Signal that we finally cached the file
	err = os.Rename(foutCaching, foutCached)
	if err != nil {
		log.Printf("Failed to rename from %s to %s", foutCaching, foutCached)
	}
	log.Printf("rename:%s -> %s", foutCaching, foutCached)
	return nil, nil
}

// CacheLocation gives the file location locally, and in the buckets
func (d *S3DrainProviderData) CacheLocation() string {
	return d.CacheLocationString
}

// CacheMustExist ensures that the cache directory exists.
func CacheMustExist(d DrainProvider) (err error) {
	if _, err = os.Stat(d.CacheLocation()); os.IsNotExist(err) {
		err = os.Mkdir(d.CacheLocation(), 0700)
		log.Printf("Creating cache directory %s", d.CacheLocation())
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
func filePurgeVisit(d *S3DrainProviderData, name string, f os.FileInfo, err error) (errReturn error) {
	if err != nil {
		log.Printf("Error walking directory for %s: %v", name, err)
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
	ext := path.Ext(name)

	//Get the current disk space usage
	sfs := syscall.Statfs_t{}
	err = syscall.Statfs(name, &sfs)
	if err != nil {
		log.Printf("Unable to purge %s, due to statfs fail:%s", name, err)
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
				errReturn := os.Remove(name)
				if errReturn != nil {
					log.Printf("Unable to purge %s", name)
				} else {
					log.Printf("Purged %s.  Age:%ds Size:%d DiskUsage:%f", name, ageInSeconds, size, usage)
				}
			}
		}
	default:
		//If something has been here for a week, and it's not cached, then it's
		//garbage.  If a machine has been turned off for a few days, the files
		//might legitimately be awaiting upload.  Other states are certainly
		//garbage after only a few hours.
		if ageInSeconds > 60*60*24*7 {
			errReturn := os.Remove(name)
			if errReturn != nil {
				msg := fmt.Sprintf("Unable to purge %s", name)
				sendAppErrorResponse(nil, NewAppError(FailPurgeAnomaly, nil, msg))
			} else {
				msg := fmt.Sprintf("Purged for age %s.  Age:%ds Size:%d DiskUsage:%f", name, ageInSeconds, size, usage)
				sendAppErrorResponse(nil, NewAppError(PurgeAnomaly, nil, msg))
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
		err = filepath.Walk(
			d.CacheLocationString,
			func(name string, f os.FileInfo, err error) (errReturn error) {
				return filePurgeVisit(d, name, f, err)
			},
		)
		if err != nil {
			log.Printf("Unable to walk cache %s: %v", d.CacheLocationString, err)
		}
		time.Sleep(d.walkSleep)
	}
}

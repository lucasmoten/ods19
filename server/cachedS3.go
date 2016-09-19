package server

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"syscall"

	globalconfig "decipher.com/object-drive-server/config"
	configx "decipher.com/object-drive-server/configx"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/uber-go/zap"
)

var (
	logger = globalconfig.RootLogger.With(zap.String("session", "drainprovider"))
	//chunksInMegs is a default number of megabytes to fetch from S3 when there is a cache miss
	chunkSize = int64(globalconfig.GetEnvOrDefaultInt("OD_AWS_S3_FETCH_MB", 16)) * int64(1024*1024)
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
func checkAWSEnvironmentVars(logger zap.Logger) {
	// Variables for the environment can be provided as either the native AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
	// or be prefixed with the common "OD_" as in OD_AWS_REGION, OD_AWS_ACCESS_KEY_ID, and OD_AWS_SECRET_ACCESS_KEY
	// Environment variables will be normalized to the AWS_ variants to facilitate internal library calls
	region := globalconfig.GetEnvOrDefault("OD_AWS_REGION", globalconfig.GetEnvOrDefault("AWS_REGION", ""))
	if len(region) > 0 {
		os.Setenv("AWS_REGION", region)
	}
	accessKeyID := globalconfig.GetEnvOrDefault("OD_AWS_ACCESS_KEY_ID", globalconfig.GetEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	if len(accessKeyID) > 0 {
		os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	}
	secretKey := globalconfig.GetEnvOrDefault("OD_AWS_SECRET_ACCESS_KEY", globalconfig.GetEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""))
	if len(secretKey) > 0 {
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	}
	// If the region is not set, then fail
	if region == "" {
		logger.Fatal("Fatal Error: Environment variable AWS_REGION must be set.")
	}
	return
}

// NullDrainProviderData is just a file location that does not talk to S3.
type NullDrainProviderData struct {
	//Where the CacheLocation is rooted on disk (ie: a very large drive mounted)
	CacheObject DrainCache
	//Location of the cache
	CacheLocationString string
	//Logger for logging
	Logger zap.Logger
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

	//Logger for logging
	Logger zap.Logger
}

// NewS3DrainProvider sets up a drain with default parameters overridden by environment variables
func NewS3DrainProvider(conf configx.S3DrainProviderOpts, name string) DrainProvider {

	walkSleepDuration := time.Duration(conf.WalkSleep) * time.Second

	d := NewS3DrainProviderRaw(conf.Root, name, conf.LowWatermark, conf.EvictAge, conf.HighWatermark, walkSleepDuration, logger)
	go d.DrainUploadedFilesToSafety()
	return d
}

// NewS3DrainProviderRaw set up a new drain provider that gives us members to use the drain and goroutine to clean cache.
// Call this to build a test cache.
func NewS3DrainProviderRaw(root, name string, lowWatermark float64, ageEligibleForEviction int64, highWatermark float64, walkSleep time.Duration, logger zap.Logger) *S3DrainProviderData {
	d := &S3DrainProviderData{
		AWSSession:             NewAWSSession(),
		CacheObject:            DrainCacheData{root},
		CacheLocationString:    name,
		lowWatermark:           lowWatermark,
		ageEligibleForEviction: ageEligibleForEviction,
		highWatermark:          highWatermark,
		walkSleep:              walkSleep,
		Logger:                 logger,
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

// Resolve a name to somewhere in the cache, given the rName
func (d *S3DrainProviderData) Resolve(fName FileName) FileNameCached {
	return FileNameCached(d.CacheLocationString + "/" + string(fName))
}

// Resolve a name to somewhere in the cache, given the rName
func (d *NullDrainProviderData) Resolve(fName FileName) FileNameCached {
	return FileNameCached(d.CacheLocationString + "/" + string(fName))
}

// NewAWSSession instantiates a connection to AWS.
func NewAWSSession() *session.Session {

	checkAWSEnvironmentVars(logger)

	region := os.Getenv("AWS_REGION")
	endpoint := os.Getenv("OD_AWS_ENDPOINT")

	// See if AWS creds in environment
	accessKeyID := globalconfig.GetEnvOrDefault("OD_AWS_ACCESS_KEY_ID", globalconfig.GetEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	secretKey := globalconfig.GetEnvOrDefault("OD_AWS_SECRET_ACCESS_KEY", globalconfig.GetEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""))
	if len(accessKeyID) > 0 && len(secretKey) > 0 {
		logger.Info("aws.credentials", zap.String("provider", "environment variables"))
		sessionConfig := &aws.Config{
			Credentials: credentials.NewEnvCredentials(),
			Region:      aws.String(region),
			Endpoint:    aws.String(endpoint),
		}
		return session.New(sessionConfig)
	}
	// Do as IAM
	logger.Info("aws.credentials", zap.String("provider", "iam role"))
	return session.New(&aws.Config{
		Region:   aws.String(region),
		Endpoint: aws.String(endpoint),
	})
}

// NewNullDrainProvider setup a drain provider that doesnt use S3 backend, just local caching.
func NewNullDrainProvider(root, name string) DrainProvider {
	d := &NullDrainProviderData{
		CacheObject:         DrainCacheData{root},
		CacheLocationString: name,
		Logger:              logger,
	}
	CacheMustExist(d, logger)
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
		d.Logger.Error(
			"unable to rename uploaded file",
			zap.String("filename", d.Files().Resolve(outFileUploaded)),
			zap.String("err", err.Error()),
		)
		return err
	}
	return nil
}

// DrainUploadedFilesToSafetyRaw is the drain without the goroutine at the end
func (d *S3DrainProviderData) DrainUploadedFilesToSafetyRaw() {
	//Walk through the cache, and handle .uploaded files
	fqCache := d.Files().Resolve(d.Resolve(""))
	err := filepath.Walk(
		fqCache,
		// We need to capture d because this interface won't let us pass it
		func(fqName string, f os.FileInfo, err error) (errReturn error) {
			if err != nil {
				sendAppErrorResponse(d.Logger, nil, NewAppError(FailCacheWalk, err,
					"error walking directory",
					zap.String("filename", fqName),
				))
				// I didn't generate this error, so I am assuming that I can just log the problem.
				// TODO: this error is not being counted
				return nil
			}

			if f.IsDir() {
				return nil
			}
			size := f.Size()
			ext := path.Ext(fqName)
			bucket := &configx.DefaultBucket
			if ext == ".uploaded" {
				fBase := path.Base(fqName)
				rName := FileId(fBase[:len(fBase)-len(ext)])
				err := d.CacheToDrain(bucket, rName, size)
				if err != nil {
					sendAppErrorResponse(d.Logger, nil, NewAppError(FailCacheToDrain, err, "error draining cache"))
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
		sendAppErrorResponse(d.Logger, nil, NewAppError(FailCacheWalk, err, "Unable to walk cache"))
	}
}

// DrainUploadedFilesToSafety moves files that were not completely sent to S3 into S3.
// This can happen if the server reboots.
func (d *S3DrainProviderData) DrainUploadedFilesToSafety() {
	d.DrainUploadedFilesToSafetyRaw()
	d.Logger.Info("cache purge start")
	//Only now can we start to purge files
	go d.CachePurge()
}

// CacheToDrain drains to S3.  Note that because this is async with respect to the http session,
// we cant return AppError.
//
// Dont delete the file here if something goes wrong... because the caller tries this multiple times
//
func (d *S3DrainProviderData) CacheToDrain(bucket *string, rName FileId, size int64) error {
	sess := d.AWSSession
	outFileUploaded := d.Resolve(FileName(rName + ".uploaded"))

	fIn, err := d.Files().Open(outFileUploaded)
	if err != nil {
		d.Logger.Error(
			"Cant drain off file",
			zap.String("filename", d.Files().Resolve(outFileUploaded)),
			zap.String("err", err.Error()),
		)
		return err
	}
	defer fIn.Close()

	key := aws.String(string(d.Resolve(NewFileName(rName, ""))))
	d.Logger.Info("drain to S3", zap.String("bucket", *bucket), zap.String("key", *key))

	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Body:   fIn,
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		d.Logger.Error(
			"Could not write to S3",
			zap.String("err", err.Error()),
		)
		return err
	}

	//Rename the file to note that it only lives here as cached for download
	//It might be deleted at any time
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
	d.Logger.Info(
		"s3 stored",
		zap.String("rname", string(rName)),
	)

	return err
}

// DrainToCache does nothing for a null drain which leaves files local.
func (d *NullDrainProviderData) DrainToCache(bucket *string, theFile FileId) (*AppError, error) {
	return nil, nil
}

// S3DownloadAttempt ...
func S3DownloadAttempt(d *S3DrainProviderData, downloader *s3manager.Downloader, fOut *os.File, bucket *string, key *string) (int64, error) {
	length, err := downloader.Download(
		fOut,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		},
	)
	if err != nil {
		d.Logger.Info("unable to download out of s3", zap.String("bucket", *bucket), zap.String("key", *key))
	}
	return length, err
}

// DrainToCache gets a file back out of the drain into the cache.
func (d *S3DrainProviderData) DrainToCache(bucket *string, rName FileId) (*AppError, error) {
	d.Logger.Info(
		"get from S3",
		zap.String("bucket", *bucket),
		zap.String("key", string(rName)),
	)

	// This file must ONLY exist for the duration of this function.
	// we must remove it or rename it before we exit.
	foutCaching := d.Resolve(NewFileName(rName, ".caching"))
	foutCached := d.Resolve(NewFileName(rName, ".cached"))
	fOut, err := d.Files().Create(foutCaching)
	if err != nil {
		msg := "unable to write local buffer"
		d.Logger.Error(
			msg,
			zap.String("filename", d.Files().Resolve(foutCaching)),
			zap.String("err", err.Error()),
		)
		sendAppErrorResponse(d.Logger, nil, NewAppError(FailDrainToCache, err, msg))
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
		_, err = S3DownloadAttempt(d, downloader, fOut, bucket, key)
		tries--
		if err == nil {
			break
		} else {
			d.Logger.Error(
				"unable to download out of S3",
				zap.String("bucket", *bucket),
				zap.Duration("seconds", waitTime/(time.Second)),
				zap.Int("more tries", tries-1),
				zap.String("key", string(rName)),
			)
			//Without a file length, this is our best guess
			time.Sleep(waitTime)
			//Fibonacci progression 1 1 2 3 ... ... 22 of them gives a total wait time of about 2 mins, or almost 8GB
			oldWaitTime := waitTime
			waitTime = prevWaitTime + waitTime
			prevWaitTime = oldWaitTime
		}
	}
	if err != nil {
		fqName := d.Files().Resolve(foutCached)
		return NewAppError(500, err, "giving up on s3 cache", zap.String("key", fqName)), err
	}

	//Signal that we finally cached the file
	err = d.Files().Rename(foutCaching, foutCached)
	if err != nil {
		d.Logger.Error(
			"rename fail",
			zap.String("from", d.Files().Resolve(foutCaching)),
			zap.String("to", d.Files().Resolve(foutCached)),
		)
	}
	d.Logger.Info("s3 fetched", zap.String("rname", string(rName)))
	return nil, nil
}

// CacheMustExist ensures that the cache directory exists.
func CacheMustExist(d DrainProvider, logger zap.Logger) (err error) {
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
func filePurgeVisit(d *S3DrainProviderData, fqName string, f os.FileInfo, err error) (errReturn error) {
	if err != nil {
		sendAppErrorResponse(d.Logger, nil, NewAppError(FailPurgeAnomaly, nil,
			"error walking directory",
			zap.String("filename", fqName),
			zap.String("err", err.Error()),
		))
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
		sendAppErrorResponse(d.Logger, nil, NewAppError(FailPurgeAnomaly, nil,
			"unable to purge on statfs fail",
			zap.String("filename", fqName),
		))
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
					sendAppErrorResponse(d.Logger, nil, NewAppError(FailPurgeAnomaly, errReturn,
						"unable to purge",
						zap.String("filename", fqName),
						zap.String("err", errReturn.Error()),
					))
					return nil
				} else {
					d.Logger.Info(
						"purge",
						zap.String("filename", fqName),
						zap.Int64("ageinseconds", ageInSeconds),
						zap.Int64("size", size),
						zap.Float64("usage", usage),
					)
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
				sendAppErrorResponse(d.Logger, nil, NewAppError(FailPurgeAnomaly, errReturn,
					"unable to purge",
					zap.String("filename", fqName),
					zap.String("err", errReturn.Error()),
				))
				return nil
			} else {
				//Count this anomaly
				sendAppErrorResponse(d.Logger, nil, NewAppError(PurgeAnomaly, nil,
					"purged for age",
					zap.String("filename", fqName),
					zap.Int64("age", ageInSeconds),
					zap.Float64("usage", usage),
				))
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
			d.Logger.Error(
				"unable to walk cache",
				zap.String("filename", fqCache),
				zap.String("err", err.Error()),
			)
		}
		time.Sleep(d.walkSleep)
	}
}

// TestS3Connection can be run to inspect the environment for configured S3
// bucket names, and verify that those buckets are writable with our credentials.
func TestS3Connection(sess *session.Session) bool {
	uploader := s3manager.NewUploader(sess)
	bucketName := globalconfig.GetEnvOrDefault("OD_AWS_S3_BUCKET", "")
	if bucketName == "" {
		logger.Error("serviceTestError",
			zap.String("err", "Missing environment variable OD_AWS_S3_BUCKET"))
		return false
	}
	input := s3.GetBucketAclInput{Bucket: strPtr(bucketName)}
	output, err := uploader.S3.GetBucketAcl(&input)
	if err != nil {
		logger.Error("serviceTestError", zap.String("err", err.Error()))
		return false
	}
	hasRead, hasWrite := false, false
	for _, grant := range output.Grants {
		if *grant.Permission == "WRITE" {
			hasWrite = true
		}
		if *grant.Permission == "READ" {
			hasRead = true
		}
	}

	if hasRead && hasWrite {
		return true
	}

	logger.Error("serviceTestError",
		zap.String("err", "Insufficient permissions on bucket"),
		zap.Object("GetBucketAclOutput", output),
	)
	return false
}

func strPtr(s string) *string { return &s }

// S3Puller hides the range requesting out of S3, to look like a file handle that just keeps giving back data
type S3Puller struct {
	S3Svc       *s3.S3
	Logger      zap.Logger
	AWSSession  *session.Session
	TotalLen    int64
	CipherStart int64
	CipherStop  int64
	Key         *string
	Bucket      *string
	StartFrom   int64
	File        io.ReadCloser
	Remaining   int64
	ChunkSize   int64
	Index       int64
}

// NewS3Puller prepares to start pulling from S3
func (d *S3DrainProviderData) NewS3Puller(logger zap.Logger, rName FileId, totalLength, cipherStartAt, cipherStopAt int64) (io.ReadCloser, error) {
	key := aws.String(string(d.Resolve(NewFileName(rName, ""))))
	bucket := &configx.DefaultBucket

	p := &S3Puller{
		S3Svc:       s3.New(d.AWSSession),
		Logger:      logger,
		AWSSession:  d.AWSSession,
		TotalLen:    totalLength,
		CipherStart: cipherStartAt,
		CipherStop:  cipherStopAt,
		Key:         key,
		Bucket:      bucket,
		File:        nil,
		ChunkSize:   chunkSize,
		Index:       cipherStartAt,
	}

	//We want to stall for first contact from S3
	sleepTime := time.Duration(1) * time.Second
	attempts := 20
	var err error
	for {
		err = p.More()
		if err == nil {
			break
		}
		attempts--
		if attempts == 0 {
			break
		}
		sleepTime = sleepTime + sleepTime
		p.Logger.Info("s3 pull stall", zap.Int("sleepInSeconds", int(sleepTime/time.Second)))
		time.Sleep(sleepTime)
	}
	return p, err
}

// More will refresh from S3 for more data
func (p *S3Puller) More() error {
	var rangeReq string
	clen := p.ChunkSize
	if p.Index == p.TotalLen {
		return io.EOF
	}
	//These numbers should have been snapped to cipher block boundaries if they were not already
	if p.Index+clen < p.TotalLen {
		rangeReq = fmt.Sprintf("bytes=%d-%d", p.Index, p.Index+clen-1)
	} else {
		clen = p.TotalLen - p.Index
		rangeReq = fmt.Sprintf("bytes=%d-", p.Index)
	}
	//p.Logger.Info("s3 fill", zap.String("range", rangeReq), zap.Int64("total", p.TotalLen))
	oOut, err := p.S3Svc.GetObject(
		&s3.GetObjectInput{
			Bucket: p.Bucket,
			Key:    p.Key,
			Range:  &rangeReq,
		},
	)
	//p.Logger.Info("s3 fill end")
	if err != nil {
		logger.Info(
			"unable to download out of s3",
			zap.String("bucket", *p.Bucket),
			zap.String("key", *p.Key),
			zap.String("err", err.Error()),
		)
		return err
	}
	p.File = oOut.Body
	p.Remaining = clen
	return nil
}

// Keep calling this to read out of S3
func (p *S3Puller) Read(data []byte) (int, error) {
	var err error
	var length int
	if p.Remaining > 0 {
		length, err = p.File.Read(data)
		//p.Logger.Info("read", zap.Int64("idx", p.Index), zap.Int64("remain", p.Remaining), zap.Int("len", length), zap.Object("err", err))
		p.Remaining -= int64(length)
		p.Index += int64(length)
	}
	if p.Remaining == 0 {
		err = p.More()
	}
	return length, err
}

// Close this when done pulling from S3
func (p *S3Puller) Close() error {
	if p.File != nil {
		//Close out this chunk from S3
		return p.File.Close()
	}
	return nil
}

// NewS3Puller isn't really used.  This makes the analogy for the S3 version clear though.
func (d *NullDrainProviderData) NewS3Puller(logger zap.Logger, rName FileId, totalLength, cipherStartAt, cipherStopAt int64) (io.ReadCloser, error) {
	cipherFile, err := d.Files().Open(d.Resolve(NewFileName(rName, ".cached")))
	cipherFile.Seek(cipherStartAt, 0)
	return cipherFile, err
}
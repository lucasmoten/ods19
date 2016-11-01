package ciphertext

import (
	"fmt"
	"io"
	"time"

	"decipher.com/object-drive-server/amazon"
	globalconfig "decipher.com/object-drive-server/config"
	configx "decipher.com/object-drive-server/configx"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/uber-go/zap"
)

var (
	// S3ChunkSize chunksInMegs is a default number of megabytes to fetch from PermanentStorage when there is a cache miss
	// For now, I am going to leave this at its default value, because for P2P and LocalStorage, it will pick a huge
	// value anyway (1GB).  This value must be small enough that when a Puller runs out of data, that whole chunk can be
	// fetched and served back in an http request before the video stream misses its deadline.  So, for video performance,
	// smaller is better.  But the smaller it is, the more S3 requests it makes (which has overhead as well).
	S3ChunkSize = int64(globalconfig.GetEnvOrDefaultInt("OD_AWS_S3_FETCH_MB", 16)) * int64(1024*1024)
)

// PermanentStorageData is where we write back permanently
type PermanentStorageData struct {
	//The bucket setting
	Bucket *string
	//The session
	AWSSession *session.Session
	//The actual S3 api
	S3 *s3.S3
}

// NewPermanentStorageData creates a place to write in to S3
func NewPermanentStorageData(sess *session.Session, bucket *string) PermanentStorage {
	return &PermanentStorageData{
		AWSSession: sess,
		S3:         s3.New(sess),
		Bucket:     bucket,
	}
}

// GetName returns a name that the permanent storage uses to identify its collection
func (s *PermanentStorageData) GetName() *string {
	return s.Bucket
}

// Upload a file into S3
func (s *PermanentStorageData) Upload(fIn io.ReadSeeker, key *string) error {
	uploader := s3manager.NewUploader(s.AWSSession)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Body:   fIn,
		Bucket: s.Bucket,
		Key:    key,
	})
	return err
}

// Download from S3
func (s *PermanentStorageData) Download(fOut io.WriterAt, key *string) (int64, error) {
	downloader := s3manager.NewDownloader(s.AWSSession)
	return downloader.Download(fOut, &s3.GetObjectInput{Bucket: s.Bucket, Key: key})
}

// GetObject from S3
func (s *PermanentStorageData) GetStream(key *string, begin, end int64) (io.ReadCloser, error) {
	var rangeReq string
	//These numbers should have been snapped to cipher block boundaries if they were not already
	if begin <= end {
		rangeReq = fmt.Sprintf("bytes=%d-%d", begin, end)
	} else {
		rangeReq = fmt.Sprintf("bytes=%d-", begin)
	}

	out, err := s.S3.GetObject(&s3.GetObjectInput{Bucket: s.Bucket, Key: key, Range: &rangeReq})
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	return out.Body, nil
}

// NewS3CiphertextCache sets up a drain with default parameters overridden by environment variables
func NewS3CiphertextCache(conf configx.S3CiphertextCacheOpts, name string) CiphertextCache {
	logger := globalconfig.RootLogger.With(zap.String("session", "CiphertextCache"))

	walkSleepDuration := time.Duration(conf.WalkSleep) * time.Second

	s3Config := configx.NewS3Config()
	sess := amazon.NewAWSSession(s3Config.AWSConfig, logger)

	//Assign permanent storage if we have a bucket name
	var permanentStorage PermanentStorage
	if configx.DefaultBucket != "" {
		permanentStorage = NewPermanentStorageData(sess, &configx.DefaultBucket)
	} else {
		logger.Info("PermanentStorage is empty because there is no bucket name")
	}

	d := NewCiphertextCacheRaw(conf.Root, name, conf.LowWatermark, conf.HighWatermark, conf.EvictAge, walkSleepDuration, S3ChunkSize, logger, permanentStorage)
	go d.DrainUploadedFilesToSafety()
	return d
}

// TestS3Connection can be run to inspect the environment for configured S3
// bucket names, and verify that those buckets are writable with our credentials.
func TestS3Connection(sess *session.Session) bool {
	logger := globalconfig.RootLogger.With(zap.String("session", "CiphertextCache"))

	uploader := s3manager.NewUploader(sess)
	bucketName := globalconfig.GetEnvOrDefault("OD_AWS_S3_BUCKET", "")
	if bucketName == "" {
		logger.Error("serviceTestError",
			zap.String("err", "Missing environment variable OD_AWS_S3_BUCKET"))
		return false
	}
	input := s3.GetBucketAclInput{Bucket: aws.String(bucketName)}
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

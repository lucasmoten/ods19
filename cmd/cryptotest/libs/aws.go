package libs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	//"log"
	//"net/http"
	"os"
	//"time"
)

func (h Uploader) awsS3(account string) (*s3.S3, *session.Session) {
	sessionConfig := &aws.Config{
		Credentials: credentials.NewSharedCredentials("", account),
	}
	sess := session.New(sessionConfig)
	svc := s3.New(sess)
	return svc, sess
}

//NewAWSBackend makes S3 backend for data storage
func (h Uploader) NewAWSBackend() *Backend {
	return &Backend{
		GetBucketReadHandle:   h.awsGetBucketReadHandle,
		GetBucketWriteHandle:  h.awsGetBucketWriteHandle,
		EnsureBucketExists:    h.awsEnsureBucketExists,
		GetBucketFileExists:   h.awsGetBucketFileExists,
		GetBucketAppendHandle: h.awsGetBucketAppendHandle,
	}
}

//Hide filesystem reads so they can be S3 buckets
func (h Uploader) awsGetBucketReadHandle(bucketKeyName string) (r io.Reader, c io.Closer, err error) {
	f, ferr := os.Open(bucketKeyName)
	return f, f, ferr
}

func (h Uploader) awsGetBucketWriteHandle(bucketKeyName string) (io.Writer, io.Closer, error) {
	f, ferr := os.Create(bucketKeyName)
	return f, f, ferr
}

func (h Uploader) awsEnsureBucketExists(bucketName string) error {
	err := os.Mkdir(bucketName, 0700)
	return err
}

func (h Uploader) awsGetBucketFileExists(bucketKeyName string) (bool, error) {
	_, err := os.Stat(bucketKeyName)

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (h Uploader) awsGetBucketAppendHandle(bucketKeyName string) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.OpenFile(bucketKeyName, os.O_RDWR|os.O_APPEND, 0600)
	return f, f, ferr
}

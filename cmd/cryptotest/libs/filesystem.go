package libs

import (
	"io"
	"os"
)

//Hide filesystem reads so they can be S3 buckets
func (h Uploader) getBucketReadHandle(bucketKeyName string) (r io.Reader, c io.Closer, err error) {
	f, ferr := os.Open(bucketKeyName)
	return f, f, ferr
}

//Hide filesystem writes so they can be S3 buckets
func (h Uploader) getBucketWriteHandle(bucketKeyName string) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.Create(bucketKeyName)
	return f, f, ferr
}

func (h Uploader) ensureBucketExists(bucketName string) error {
	err := os.Mkdir(bucketName, 0700)
	return err
}

func (h Uploader) getBucketFileExists(bucketKeyName string) (bool, error) {
	_, err := os.Stat(bucketKeyName)

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (h Uploader) getBucketAppendHandle(bucketKeyName string) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.OpenFile(bucketKeyName, os.O_RDWR|os.O_APPEND, 0600)
	return f, f, ferr
}

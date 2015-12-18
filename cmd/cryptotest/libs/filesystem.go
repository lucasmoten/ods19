package libs

import (
	"io"
	"os"
)

//NewFilesystemBackend creates a backend backend object that runs against the filesystem
func (h Uploader) NewFilesystemBackend() *Backend {
	return &Backend{
		GetBucketReadHandle:   h.filesystemGetBucketReadHandle,
		GetBucketWriteHandle:  h.filesystemGetBucketWriteHandle,
		EnsureBucketExists:    h.filesystemEnsureBucketExists,
		GetBucketFileExists:   h.filesystemGetBucketFileExists,
		GetBucketAppendHandle: h.filesystemGetBucketAppendHandle,
	}
}

//Hide filesystem reads so they can be S3 buckets
func (h Uploader) filesystemGetBucketReadHandle(bucketKeyName string) (r io.Reader, c io.Closer, err error) {
	f, ferr := os.Open(bucketKeyName)
	return f, f, ferr
}

//Hide filesystem writes so they can be S3 buckets
func (h Uploader) filesystemGetBucketWriteHandle(bucketKeyName string) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.Create(bucketKeyName)
	return f, f, ferr
}

func (h Uploader) filesystemEnsureBucketExists(bucketName string) error {
	err := os.Mkdir(bucketName, 0700)
	return err
}

func (h Uploader) filesystemGetBucketFileExists(bucketKeyName string) (bool, error) {
	_, err := os.Stat(bucketKeyName)

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (h Uploader) filesystemGetBucketAppendHandle(bucketKeyName string) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.OpenFile(bucketKeyName, os.O_RDWR|os.O_APPEND, 0600)
	return f, f, ferr
}

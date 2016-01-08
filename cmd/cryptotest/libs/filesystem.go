package libs

import (
	"io"
	"os"
)

//NewFilesystemBackend creates a backend backend object that runs against the filesystem
func (h Uploader) NewFilesystemBackend() *Backend {
	return &Backend{
		GetReadHandle:         h.filesystemGetReadHandle,
		GetWriteHandle:        h.filesystemGetWriteHandle,
		EnsurePartitionExists: h.filesystemEnsurePartitionExists,
		GetFileExists:         h.filesystemGetFileExists,
		GetAppendHandle:       h.filesystemGetAppendHandle,
		DeleteFile:            h.filesystemDeleteFile,
	}
}

func (h Uploader) filesystemDeleteFile(bucketKeyName string) error {
	return os.Remove(bucketKeyName)
}

//Hide filesystem reads so they can be S3 buckets
func (h Uploader) filesystemGetReadHandle(bucketKeyName string) (r io.Reader, c io.Closer, err error) {
	f, ferr := os.Open(bucketKeyName)
	return f, f, ferr
}

//Hide filesystem writes so they can be S3 buckets
func (h Uploader) filesystemGetWriteHandle(bucketKeyName string) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.Create(bucketKeyName)
	return f, f, ferr
}

func (h Uploader) filesystemEnsurePartitionExists(bucketName string) error {
	err := os.Mkdir(bucketName, 0700)
	return err
}

func (h Uploader) filesystemGetFileExists(bucketKeyName string) (bool, error) {
	_, err := os.Stat(bucketKeyName)

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (h Uploader) filesystemGetAppendHandle(bucketKeyName string) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.OpenFile(bucketKeyName, os.O_RDWR|os.O_APPEND, 0600)
	return f, f, ferr
}

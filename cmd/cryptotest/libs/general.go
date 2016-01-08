package libs

import (
	"io"
)

/*Uploader is a special type of Http server.
  Put any config state in here.
  The point of this server is to show how
  upload and download can be extremely efficient
  for large files.
*/
type Uploader struct {
	Partition      string
	Port           int
	Bind           string
	Addr           string
	UploadCookie   string
	BufferSize     int
	KeyBytes       int
	RSAEncryptBits int
	Backend        *Backend
	Tracker        *JobReporters
}

//Backend can be implemented as S3, filesystem, etc
type Backend struct {
	GetReadHandle         func(fileName string) (r io.Reader, c io.Closer, err error)
	GetWriteHandle        func(fileName string) (w io.Writer, c io.Closer, err error)
	EnsurePartitionExists func(fileName string) error
	GetFileExists         func(fileName string) (bool, error)
	GetAppendHandle       func(fileName string) (w io.Writer, c io.Closer, err error)
	DeleteFile            func(fileName string) error
}

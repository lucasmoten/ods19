package libs

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
)

type fileDirPath string
type bindIPAddr string
type bindURL string

/*Uploader is a special type of Http server.
  Put any config state in here.
  The point of this server is to show how
  upload and download can be extremely efficient
  for large files.
*/
type Uploader struct {
	HomeBucket     fileDirPath
	Port           int
	Bind           bindIPAddr
	Addr           bindURL
	UploadCookie   string
	BufferSize     int
	KeyBytes       int
	RSAEncryptBits int
	Session        *s3.S3 //??? should this not be global due to locks???
	Backend        *Backend
}

//Backend can be implemented as S3, filesystem, etc
type Backend struct {
	GetBucketReadHandle   func(bucketKeyName string) (r io.Reader, c io.Closer, err error)
	GetBucketWriteHandle  func(bucketKeyName string) (w io.Writer, c io.Closer, err error)
	EnsureBucketExists    func(bucketName string) error
	GetBucketFileExists   func(bucketKeyName string) (bool, error)
	GetBucketAppendHandle func(bucketKeyName string) (w io.Writer, c io.Closer, err error)
}

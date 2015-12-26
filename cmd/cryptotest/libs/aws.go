package libs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
	//"net/http"
	"crypto/aes"
	"crypto/sha256"
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
		GetReadHandle:         h.awsGetReadHandle,
		GetWriteHandle:        h.awsGetWriteHandle,
		EnsurePartitionExists: h.awsEnsurePartitionExists,
		GetFileExists:         h.awsGetFileExists,
		GetAppendHandle:       h.awsGetAppendHandle,
	}
}

//Hide filesystem reads so they can be S3 buckets
func (h Uploader) awsGetReadHandle(
	bucketKeyName string,
) (r io.Reader, c io.Closer, err error) {
	f, ferr := os.Open(h.Partition + "/" + bucketKeyName)
	return f, f, ferr
}

func (h Uploader) awsGetWriteHandle(
	bucketKeyName string,
) (io.Writer, io.Closer, error) {
	f, ferr := os.Create(h.Partition + "/" + bucketKeyName)
	return f, f, ferr
}

func (h Uploader) awsEnsurePartitionExists(bucketName string) error {
	err := os.Mkdir(bucketName, 0700)
	return err
}

func (h Uploader) awsGetFileExists(bucketKeyName string) (bool, error) {
	_, err := os.Stat(h.Partition + "/" + bucketKeyName)

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (h Uploader) awsGetAppendHandle(
	bucketKeyName string,
) (w io.Writer, c io.Closer, err error) {
	f, ferr := os.OpenFile(h.Partition+"/"+bucketKeyName, os.O_RDWR|os.O_APPEND, 0600)
	return f, f, ferr
}

func (h Uploader) drainFileToS3(
	svc *s3.S3,
	sess *session.Session,
	bucket *string,
	fName string,
) error {
	fIn, err := os.Open(h.Partition + "/" + fName)
	if err != nil {
		log.Printf("Cant drain off file: %v", err)
		return err
	}
	defer fIn.Close()
	log.Printf("draining to S3 %s: %s", *bucket, fName)

	uploader := s3manager.NewUploader(sess)
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   fIn,
		Bucket: bucket,
		Key:    aws.String(h.Partition + "/" + fName),
	})
	if err != nil {
		log.Printf("Could not write to S3: %v", err)
		return err
	}
	log.Printf("Uploaded to %v: %v", *bucket, result.Location)
	return err
}

func (h Uploader) drainToS3(
	keyName,
	keyFileName,
	ivFileName,
	classFileName,
	checksumFileName string,
) error {
	var err error
	svc, sess := h.awsS3(awsConfig)
	bucket := aws.String(awsBucket)
	h.drainFileToS3(svc, sess, bucket, keyName)
	h.drainFileToS3(svc, sess, bucket, keyFileName)
	h.drainFileToS3(svc, sess, bucket, ivFileName)
	h.drainFileToS3(svc, sess, bucket, classFileName)
	h.drainFileToS3(svc, sess, bucket, checksumFileName)
	return err
}

func (h Uploader) transferFileFromS3(
	svc *s3.S3,
	sess *session.Session,
	bucket *string,
	theFile string,
) {
	log.Printf("Get from S3 bucket %s: %s", *bucket, theFile)

	fOut, err := os.Create(h.Partition + "/" + theFile)
	if err != nil {
		log.Printf("Unable to write local buffer file %s: %v", theFile, err)
	}
	defer fOut.Close()

	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(
		fOut,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    aws.String(h.Partition + "/" + theFile),
		},
	)
	if err != nil {
		log.Printf("Unable to download out of S3 bucket %v: %v", *bucket, theFile)
	}
}

//Ensure that we get copies on the filesystem from S3
func (h Uploader) transferFromS3(fName, dn string) {
	fNameData := fName + ".data"
	fNameKey := dn + "/" + fName + ".key"
	fNameIV := fName + ".iv"
	fNameClass := fName + ".class"
	fNameHash := fName + ".hash"

	svc, sess := h.awsS3(awsConfig)
	bucket := aws.String(awsBucket)

	h.Backend.EnsurePartitionExists(h.Partition + "/" + dn)

	h.transferFileFromS3(svc, sess, bucket, fNameData)
	h.transferFileFromS3(svc, sess, bucket, fNameKey)
	h.transferFileFromS3(svc, sess, bucket, fNameIV)
	h.transferFileFromS3(svc, sess, bucket, fNameClass)
	h.transferFileFromS3(svc, sess, bucket, fNameHash)
}

func (h Uploader) retrieveChecksumData(fileName string) (checksum []byte, err error) {
	checksumFileName := fileName + ".hash"
	checksumFile, closer, err := h.Backend.GetReadHandle(checksumFileName)
	if err != nil {
		return checksum, err
	}
	defer closer.Close()
	checksum = make([]byte, sha256.BlockSize)
	checksumFile.Read(checksum)
	return checksum, err
}

func (h Uploader) retrieveMetaData(fileName string, dn string) (key []byte, iv []byte, cls []byte, err error) {
	userDir := obfuscateHash(dn)
	keyFileName := userDir + "/" + fileName + ".key"
	ivFileName := fileName + ".iv"
	classFileName := fileName + ".class"

	classFile, closer, err := h.Backend.GetReadHandle(classFileName)
	if err != nil {
		return key, iv, cls, err
	}
	defer closer.Close()
	cls = make([]byte, 80)
	classFile.Read(cls)

	keyFile, closer, err := h.Backend.GetReadHandle(keyFileName)
	if err != nil {
		return key, iv, cls, err
	}
	defer closer.Close()
	key = make([]byte, h.KeyBytes)
	keyFile.Read(key)

	ivFile, closer, err := h.Backend.GetReadHandle(ivFileName)
	if err != nil {
		return key, iv, cls, err
	}
	defer closer.Close()
	iv = make([]byte, aes.BlockSize)
	ivFile.Read(iv)

	applyPassphrase([]byte(masterKey), key)
	return key, iv, cls, nil
}

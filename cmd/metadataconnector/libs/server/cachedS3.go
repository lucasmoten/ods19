package server

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	CacheLocationString string
}

// S3DrainProviderData moves data from cache to the drain... S3 buckets in this case.
type S3DrainProviderData struct {
	CacheLocationString string
	AWSSession          *session.Session
}

// NewS3DrainProvider set up a new drain provider that gives us members to use the drain and goroutine to clean cache.
func NewS3DrainProvider(name string) DrainProvider {
	checkAWSEnvironmentVars()

	d := &S3DrainProviderData{
		AWSSession:          awsS3(),
		CacheLocationString: name,
	}
	CacheMustExist(d)
	go d.DrainUploadedFilesToSafety()
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
				log.Printf("Error walking directory on initial upload for %s: %v", name, err)
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
					log.Printf("error draining cache:%v", err)
				}
			}
			return nil
		},
	)
	if err != nil {
		log.Printf("Unable to walk cache %s: %v", d.CacheLocationString, err)
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
	foutCaching := d.CacheLocationString + "/" + theFile + ".caching"
	foutCached := d.CacheLocationString + "/" + theFile + ".cached"
	fOut, err := os.Create(foutCaching)
	if err != nil {
		log.Printf("Unable to write local buffer file %s: %v", theFile, err)
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
	}
	//Signal that we finally cached the file
	err = os.Rename(foutCaching, foutCached)
	if err != nil {
		log.Printf("Failed to rename from %s to %s", foutCaching, foutCached)
	}
	log.Printf("rename:%s -> %s", foutCaching, foutCached)
	return nil, nil
}

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
func filePurgeVisit(name string, f os.FileInfo, err error) (errReturn error) {
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

	t := f.ModTime().Unix() //In units of second
	n := time.Now().Unix()  //In unites of second
	ageInSeconds := n - t
	size := f.Size()
	ext := path.Ext(name)

	switch {
	case ext == ".cached":
		/**
						  Simple purging scheme:
						    a file must be older than a minute since last use
						    its (integer) value is its size divided by age squared
						    when its value is zero, get rid of it.

						    this does NOT take into account the available space,
						    nor the insert rate.  it is rather agressive though.
				        Example values:
				        10GB 1day - 1
				         6GB 1day - 0
				       500MB 4hrs - 2
		             1MB 15mins - 1
		             1MB 20mins - 0
		*/
		if ageInSeconds > 60 {
			value := size / (ageInSeconds * ageInSeconds)
			if value == 0 {
				errReturn := os.Remove(name)
				if errReturn != nil {
					log.Printf("Unable to purge %s", name)
				} else {
					log.Printf("Purged %s.  Age:%ds Size:%d", name, ageInSeconds, size)
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
				log.Printf("Unable to purge %s", name)
			} else {
				log.Printf("Purged %s.  Age:%ds Size:%d", name, ageInSeconds, size)
			}
		}
	}
	return
}

// CachePurge will periodically delete files that do not need to be in the cache.
func (d *S3DrainProviderData) CachePurge() {
	for {
		err := filepath.Walk(d.CacheLocationString, filePurgeVisit)
		if err != nil {
			log.Printf("Unable to walk cache %s: %v", d.CacheLocationString, err)
		}
		time.Sleep(30 * time.Second)
	}
}

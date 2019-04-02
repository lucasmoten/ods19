package ciphertext

import (
	"io"
	"os"
	"path/filepath"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/util"

	"go.uber.org/zap"
)

// PermanentStorageLocalData is where we write back permanently
type PermanentStorageLocalData struct {
	// The location on disk
	Location string
}

// NewPermanentStorageLocalData creates a place to write in to S3
func NewPermanentStorageLocalData(location string) PermanentStorage {
	return &PermanentStorageLocalData{
		Location: location,
	}
}

// GetName returns a name that the permanent storage uses to identify its collection
func (s *PermanentStorageLocalData) GetName() *string {
	return &s.Location
}

// Upload a file into PermanentStorage
func (s *PermanentStorageLocalData) Upload(fIn io.ReadSeeker, key *string) error {
	fName := s.Location + "/" + *key
	fOut, err := os.Create(fName)
	if err != nil {
		return err
	}
	defer fOut.Close()
	_, err = io.Copy(fOut, fIn)
	return err
}

// Download from PermanentStorage io.WriteAt is used because there is some parallel download stuff going on with S3
func (s *PermanentStorageLocalData) Download(fOut io.WriterAt, key *string) (int64, error) {
	fName := s.Location + "/" + *key
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return 0, util.NewLoggable(PermanentStorageNotFoundErrorString, err)
	}
	fIn, err := os.Open(fName)
	if err != nil {
		return int64(0), err
	}
	defer fIn.Close()
	readBuffer := make([]byte, 1024)
	at := int64(0)
	for {
		n, err := fIn.Read(readBuffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return at, err
		}
		fOut.WriteAt(readBuffer[0:n], at)
		at += int64(n)
	}
	return at, err
}

// GetStream from PermanentStorage
func (s *PermanentStorageLocalData) GetStream(key *string, begin, end int64) (io.ReadCloser, error) {
	fName := s.Location + "/" + *key
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return nil, util.NewLoggable(PermanentStorageNotFoundErrorString, err)
	}
	fIn, err := os.Open(fName)
	if err != nil {
		return nil, err
	}
	_, err = fIn.Seek(begin, 0)
	if err != nil {
		return nil, err
	}
	return fIn, err
}

// NewLocalCiphertextCache sets up a cache with default parameters overridden by environment variables
func NewLocalCiphertextCache(logger *zap.Logger, zone CiphertextCacheZone, conf config.DiskCacheOpts, dbID string) (*CiphertextCacheData, *util.Loggable) {
	//Create a permanent storage with all directories in it pre-created
	var permanentStorage PermanentStorage
	// This is S3 behavior - done before every insert.  Imitate it here.
	at := filepath.Join(conf.Root, "permanent")
	os.MkdirAll(filepath.Join(at, conf.Partition, dbID), 0700)
	permanentStorage = NewPermanentStorageLocalData(at)
	logger.Info("PermanentStorage create", zap.String("location", at))
	d, err := NewCiphertextCacheRaw(zone, &conf, dbID, logger, permanentStorage)
	return d, err
}

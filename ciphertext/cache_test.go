package ciphertext_test

import (
	"io"
	"os"
	"testing"

	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/util"

	cfg "decipher.com/object-drive-server/config"
)

func cacheParams(root, partition string) (string, ciphertext.CiphertextCacheZone, config.S3CiphertextCacheOpts, string) {
	// Ensure that uses of decryptor will succeed
	os.Setenv(config.OD_TOKENJAR_LOCATION, "../defaultcerts/token.jar")

	masterKey := "testkey"
	chunkSize16MB := int64(16 * 1024 * 1024)
	conf := config.S3CiphertextCacheOpts{
		Root:          root,
		Partition:     partition,
		LowWatermark:  float64(0.50),
		HighWatermark: float64(0.75),
		EvictAge:      int64(60 * 5),
		WalkSleep:     120,
		ChunkSize:     chunkSize16MB,
		MasterKey:     masterKey,
	}
	at := conf.Root + "/" + conf.Partition
	dbID := "dbtest"
	zone := ciphertext.S3_DEFAULT_CIPHERTEXT_CACHE
	return at, zone, conf, dbID
}

//
// Write a non-trivial file into cache
//
func TestCacheSimple(t *testing.T) {
	logger := cfg.RootLogger
	testRoot := os.TempDir()
	testPartition := "partition0"
	_, zone, conf, dbID := cacheParams(testRoot, testPartition)
	var loggableErr *util.Loggable

	// Create the cache successfully
	d, loggableErr := ciphertext.NewLocalCiphertextCache(logger, zone, conf, dbID)
	if loggableErr != nil {
		loggableErr.ToError(logger)
		t.Logf("unable to create cache")
		t.FailNow()
	}
	t.Logf("initially created a cache. canary was Writeback to PermanentStorage.")

	t.Logf("reading source to write into cache")
	fIn, err := os.Open("./cache_test.go")
	if err != nil {
		t.Logf("unable to open source file for testing: %v", err)
		t.FailNow()
	}
	rName := ciphertext.FileId("sourcetest")
	fNameUploaded := d.Resolve(ciphertext.NewFileName(rName, ".uploaded"))

	fOut, err := d.Files().Create(fNameUploaded)
	if err != nil {
		t.Logf("unable to create source file for testing: %v", err)
		t.FailNow()
	}
	size, err := io.Copy(fOut, fIn)
	if err != nil {
		t.Logf("unable to upload source file for testing: %v", err)
		t.FailNow()
	}
	t.Logf("uploaded source to write into cache")

	err = d.Writeback(rName, size)
	if err != nil {
		t.Logf("unable to writeback file for testing: %v", err)
		t.FailNow()
	}
	t.Logf("wrote back source to PermanentStorage")

	fNameCached := d.Resolve(ciphertext.NewFileName(rName, ".cached"))
	err = d.Files().Remove(fNameCached)
	if err != nil {
		t.Logf("unable to purge file from cache: %v", err)
		t.FailNow()
	}

	d.Recache(rName)
	if _, err = d.Files().Stat(fNameCached); os.IsNotExist(err) {
		t.Logf("recache failed: %v", err)
		t.FailNow()
	}
	t.Logf("recached file")
}

//
// Create a (local - not S3!) cache.
// Bring it up with the initial key, then bring up with wrong key, then correct key.
//
func TestCacheCreateWrongKey(t *testing.T) {
	logger := cfg.RootLogger
	testRoot := os.TempDir()
	testPartition := "partition0"
	_, zone, conf, dbID := cacheParams(testRoot, testPartition)
	var loggableErr *util.Loggable

	// Create the cache successfully
	d, loggableErr := ciphertext.NewLocalCiphertextCache(logger, zone, conf, dbID)
	if loggableErr != nil {
		loggableErr.ToError(logger)
		t.Logf("unable to create cache")
		t.FailNow()
	}
	t.Logf("initially created a cache. canary was Writeback to PermanentStorage.")

	// Bring up cache code over existing cache directory with wrong key - where it's cached
	wrongKey := "wrongKey"
	correctKey := conf.MasterKey
	conf.MasterKey = wrongKey
	d, loggableErr = ciphertext.NewLocalCiphertextCache(logger, zone, conf, dbID)
	if loggableErr == nil {
		t.Logf("we used wrong key, and should have not gotten a cache")
		t.FailNow()
	}
	t.Logf("wrong key correctly did not return a cache")

	// Bring up cache code over existing cache directory with wrong key - where it's purged
	// Delete the local cache - not the permanent storage
	//
	// Note: this tests that Writeback and Recache are working
	//
	d.Delete()
	conf.MasterKey = wrongKey
	d, loggableErr = ciphertext.NewLocalCiphertextCache(logger, zone, conf, dbID)
	if loggableErr == nil {
		t.Logf("we used wrong key, and should have not gotten a cache")
		t.FailNow()
	}
	t.Logf("wrong key from purged cache correctly did not return a cache. Was compared with Recache canary from PermanentStorage.")

	// Bring up cache code over existing cache directory with correct key
	conf.MasterKey = correctKey
	d, loggableErr = ciphertext.NewLocalCiphertextCache(logger, zone, conf, dbID)
	if loggableErr != nil {
		loggableErr.ToError(logger)
		t.Logf("unable to create cache second time")
		t.FailNow()
	}
	t.Logf("correct key correctly brought up existing cache")

	// Bring up cache code over existing purged cache directory with correct key
	d.Delete()
	conf.MasterKey = correctKey
	d, loggableErr = ciphertext.NewLocalCiphertextCache(logger, zone, conf, dbID)
	if loggableErr != nil {
		loggableErr.ToError(logger)
		t.Logf("unable to create cache second time")
		t.FailNow()
	}
	t.Logf("correct key correctly brought up existing purged cache")

}

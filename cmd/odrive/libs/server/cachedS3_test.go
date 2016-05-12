package server_test

import (
	"os"
	"testing"

	"decipher.com/object-drive-server/cmd/odrive/libs/server"
)

func TestCacheCreate(t *testing.T) {
	//Setup and teardown
	bucket := "decipherers"
	dirname := "t01234"
	//Create raw cache without starting the purge goroutine
	d := server.NewS3DrainProviderRaw(".", dirname, float64(0.50), int64(60*5), float64(0.75), 120)

	//create a small file
	rName := server.FileId("fark")
	uploadedName := d.Resolve(server.NewFileName(rName, ".uploaded"))
	fqUploadedName := d.Files().Resolve(uploadedName)
	//we create the file in uploaded state
	f, err := d.Files().Create(uploadedName)
	if err != nil {
		t.Errorf("Could not create file %s:%v", fqUploadedName, err)
	}

	//cleanup
	defer f.Close()
	defer func() {
		err := d.Files().RemoveAll(server.FileNameCached(dirname))
		if err != nil {
			t.Errorf("Could not remove directory %s:%v", dirname, err)
		}
	}()

	fdata := []byte("hello world!")
	//put bytes into small file
	_, err = f.Write(fdata)
	if err != nil {
		t.Errorf("could not write to %s:%v", fqUploadedName, err)
	}

	//Write it to S3
	err = d.CacheToDrain(&bucket, rName, int64(len(fdata)))
	if err != nil {
		t.Errorf("Could not cache to drain:%v", err)
	}
	//Delete it from cache manually
	cachedName := d.Resolve(server.NewFileName(rName, ".cached"))
	err = d.Files().Remove(cachedName)
	if err != nil {
		t.Errorf("Could not remove cached file:%v", err)
	}

	//See if it is pulled from S3 properly
	herr, err := d.DrainToCache(&bucket, rName)
	if err != nil {
		t.Errorf("Could not drain to cache:%v", err)
	}
	if herr != nil {
		t.Errorf("Could not drain to cache:%v", herr)
	}
	cachingName := d.Resolve(server.NewFileName(rName, ".caching"))
	if _, err = d.Files().Stat(cachingName); os.IsNotExist(err) == false {
		t.Errorf("caching file should be removed:%v", err)
	}
	if _, err = d.Files().Stat(cachedName); os.IsExist(err) {
		t.Errorf("cached file shoud exist:%v", err)
	}

	//Read the file back and verify same content
	f, err = d.Files().Open(cachedName)
	defer f.Close()
	buf := make([]byte, 256)
	lread, err := f.Read(buf)
	if err != nil {
		t.Errorf("unable to read file:%v", err)
	}
	s1 := string(fdata)
	s2 := string(buf)[:lread]
	if s1 != s2 {
		t.Errorf("content did not come back as same values. %s vs %s", s1, s2)
	}
}

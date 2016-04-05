package server_test

import (
	"os"
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/server"
)

func TestCacheCreate(t *testing.T) {
	//Setup and teardown
	bucket := "decipherers"
	dirname := "t01234"
	err := os.Mkdir(dirname, 0700)
	if err != nil {
		t.Errorf("Could not create directory %s:%v", dirname, err)
	}

	//create a small file
	rname := "fark"
	fname := dirname + "/" + rname
	//we create the file in uploaded state
	f, err := os.Create(fname + ".uploaded")
	if err != nil {
		t.Errorf("Could not create file %s:%v", fname, err)
	}

	//cleanup
	defer f.Close()
	defer func() {
		err := os.RemoveAll(dirname)
		if err != nil {
			t.Errorf("Could not remove directory %s:%v", dirname, err)
		}
	}()

	fdata := []byte("hello world!")
	//put bytes into small file
	_, err = f.Write(fdata)
	if err != nil {
		t.Errorf("could not write to %s:%v", fname, err)
	}

	//Create raw cache without starting the purge goroutine
	d := server.NewS3DrainProviderRaw(dirname, float64(0.50), int64(60*5), float64(0.75), 120)

	//Write it to S3
	err = d.CacheToDrain(&bucket, rname, int64(len(fdata)))
	if err != nil {
		t.Errorf("Could not cache to drain:%v", err)
	}
	//Delete it from cache manually
	err = os.Remove(fname + ".cached")
	if err != nil {
		t.Errorf("Could not remove cached file:%v", err)
	}

	//See if it is pulled from S3 properly
	herr, err := d.DrainToCache(&bucket, rname)
	if err != nil {
		t.Errorf("Could not drain to cache:%v", err)
	}
	if herr != nil {
		t.Errorf("Could not drain to cache:%v", herr)
	}
	if _, err = os.Stat(fname + ".caching"); os.IsNotExist(err) == false {
		t.Errorf("caching file should be removed:%v", err)
	}
	if _, err = os.Stat(fname + ".cached"); os.IsNotExist(err) {
		t.Errorf("cached file shoud exist:%v", err)
	}

	//Read the file back and verify same content
	f, err = os.Open(fname + ".cached")
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

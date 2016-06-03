package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	s3 "github.com/rlmcpherson/s3gof3r"
)

const mb int64 = 1024

// DownloadFromS3 gets the targeted key from an S3 bucket
func DownloadFromS3(bucketName, key string) error {

	client := getS3ClientFromEnv()
	b := client.Bucket(bucketName)
	r, h, err := b.GetReader(key, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Attempting download for key with these headers:", h)

	// create file handle
	f, err := os.Create(key)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

// MoveToS3 will move to s3 bucket
func MoveToS3(path, bucketName, key string) error {

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		val := os.Getenv("OD_AWS_ACCESS_KEY_ID")
		os.Setenv("AWS_ACCESS_KEY_ID", val)
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		val := os.Getenv("OD_AWS_SECRET_ACCESS_KEY")
		os.Setenv("AWS_SECRET_ACCESS_KEY", val)
	}

	keys, err := s3.EnvKeys()
	if err != nil {
		log.Fatal(err)
	}

	timeout := time.Second * 5

	// Tweak Concurrency and PartSize to adjust the max
	// memory the Go process will consume.
	// Concurrency: 5 + PartSize: 10 * 1024 yielded a max
	// memory consumption of ~ 40 MB during 1GB upload
	uploadConfig := &s3.Config{
		Concurrency: 5,
		PartSize:    10 * mb,
		NTry:        10,
		Md5Check:    true,
		Scheme:      "https",
		Client:      s3.ClientWithTimeout(timeout),
	}

	client := s3.New("", keys)
	b := client.Bucket(bucketName)
	w, err := b.PutWriter(key, nil, uploadConfig)
	if err != nil {
		return err
	}
	defer w.Close()

	localFile := path
	f, _ := os.Open(localFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// copy to writer from file (an io.Reader)
	// Read about io.Copy and io.Reader and io.Writer
	n, err := io.Copy(w, f)
	if err != nil {
		return err
	}

	log.Printf("Total bytes written: %v \n", n)

	return nil
}

func getS3ClientFromEnv() *s3.S3 {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		val := os.Getenv("OD_AWS_ACCESS_KEY_ID")
		os.Setenv("AWS_ACCESS_KEY_ID", val)
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		val := os.Getenv("OD_AWS_SECRET_ACCESS_KEY")
		os.Setenv("AWS_SECRET_ACCESS_KEY", val)
	}

	keys, err := s3.EnvKeys()
	if err != nil {
		log.Fatal(err)
	}

	client := s3.New("", keys)
	return client
}

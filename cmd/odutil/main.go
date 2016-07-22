package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rlmcpherson/s3gof3r"
)

var (
	bucket = flag.String("bucket", "odrive-builds", "The bucket to target.")
	input  = flag.String("input", "", "input file")
	cmd    = flag.String("cmd", "upload", "Command to run.")
	key    = flag.String("key", "", "Key up on S3 to write. defaults to the value passed to input")
)

const (
	upload   = "upload"
	download = "download"
)

func main() {
	s3gof3r.SetLogger(os.Stdout, "ODUTIL ", 0, true)

	flag.Parse()
	switch *cmd {
	case upload:
		uploadRoutine(*input, *bucket, *key)
	case download:
		downloadRoutine(*input, *bucket, *key)
	default:
		fmt.Println("Unrecognized command:", *cmd)
	}

}

func uploadRoutine(input, bucket, s3key string) {
	key := formatKey(input, s3key)
	MoveToS3(input, bucket, key)
}

func downloadRoutine(input, bucket, s3key string) {
	key := formatKey(input, s3key)
	DownloadFromS3(bucket, input, key)
}

func formatKey(input, s3key string) string {
	_, key := filepath.Split(input)
	if s3key != "" {
		key = s3key
	}
	return key
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

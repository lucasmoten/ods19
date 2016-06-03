package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	bucket = flag.String("bucket", "odrive-builds", "The bucket to target.")
	input  = flag.String("input", "", "input file")
	cmd    = flag.String("cmd", "upload", "Command to run.")
)

const (
	upload   = "upload"
	download = "download"
)

func main() {

	flag.Parse()
	switch *cmd {
	case upload:
		uploadRoutine(*input, *bucket)
	case download:
		downloadRoutine(*input, *bucket)
	default:
		fmt.Println("Unrecognized command:", *cmd)
	}

}

func uploadRoutine(input, bucket string) {
	// key will become the "filename" you see in the s3 bucket. Passing
	// an entire filepath as the key will yield nested directory structures
	// on s3 itself.
	_, key := filepath.Split(input)
	fmt.Printf("Uploading file %s to bucket %s\n", input, bucket)
	fmt.Printf("Extracted filename from input: %s\n", key)
	MoveToS3(input, bucket, key)
}

func downloadRoutine(input, bucket string) {
	DownloadFromS3(bucket, input)
}

// helper functions
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

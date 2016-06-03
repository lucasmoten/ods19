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
)

func main() {

	// Invoke this tool like so:
	// ./odutil -input README.md -bucket odrive-builds

	flag.Parse()
	fmt.Printf("Uploading file %s to bucket %s\n", *input, *bucket)
	// key will become the "filename" you see in the s3 bucket. Passing
	// an entire filepath as the key will yield nested directory structures
	// on s3 itself.
	_, key := filepath.Split(*input)
	fmt.Printf("Extracted filename from input: %s\n", key)
	MoveToS3(*input, *bucket, key)
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

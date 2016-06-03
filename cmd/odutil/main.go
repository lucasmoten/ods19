package main

import (
	"flag"
	"fmt"
	"os"
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
	MoveToS3(*input, *bucket, *input)
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

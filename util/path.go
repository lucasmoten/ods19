package util

import "os"

// PathExists will return a boolean indicating if the file or directory
// exists. Returns error if something more catastrophic occurs.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

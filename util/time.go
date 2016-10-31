package util

import "time"

// NowMS is in units of milliseconds
func NowMS() int64 {
	return (time.Now().UnixNano() / (1000 * 1000))
}

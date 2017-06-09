package util

import (
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

// NowMS is in units of milliseconds
func NowMS() int64 {
	return (time.Now().UnixNano() / (1000 * 1000))
}

// Time a function - Defer the returned function to time from now until the defer completes
func Time(name string) func() {
	beginTSInMS := NowMS()
	return func() {
		interval := time.Duration(NowMS()-beginTSInMS) * time.Millisecond
		t := metrics.GetOrRegisterTimer(name, metrics.DefaultRegistry)
		t.Update(interval)
	}
}

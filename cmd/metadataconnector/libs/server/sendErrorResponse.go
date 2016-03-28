package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

var (
	// The counters for error codes
	counters      map[counterKey]int64
	countersIn    chan counterKey
	writeRequests chan http.ResponseWriter
	writeDone     chan int
)

// sendErrorResponse is the publicly called function for sending error response from top level handlers
func (h AppServer) sendErrorResponse(w http.ResponseWriter, code int, err error, msg string) {
	pc, file, line, _ := runtime.Caller(1)
	endpointParts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	endpoint := endpointParts[len(endpointParts)-1]

	log.Printf("httpCode %d:%s at %s:%d %s:%v", code, endpoint, file, line, msg, err)
	if countersIn != nil {
		countersIn <- counterKey{code, endpoint, file, line}
	}
	http.Error(w, msg, code)
	return
}

// writeCounters lets us write the counters out to stats
func (h AppServer) renderErrorCounters(w http.ResponseWriter) {
	doWriteCounters(w)
}

/*
  Error counters keep a matrix of {errorCode,endpoint} like:
    200,createObject
    500,deleteObject

  This is part of cleaning up logging and error handling
*/

// We key counters by code and endpoint tuple
type counterKey struct {
	Code     int
	Endpoint string
	//file:line are not necessarily required, but they do help to isolate exactly which code location
	File string
	Line int
}

// CounterRoutine is the goroutine that absorbs counts
func counterRoutine() {
	log.Printf("Begin counter routine")
	for {
		select {
		case key := <-countersIn:
			counters[key]++
		}
	}
}

// Write the counters out.  Make sure we are in the thread of the datastructure when we do this
func doWriteCounters(w http.ResponseWriter) {
	fmt.Fprintf(w, "count code endpoint file:line\n")
	for k, v := range counters {
		fmt.Fprintf(w, "%d %d %s %s:%d\n", v, k.Code, k.Endpoint, k.File, k.Line)
	}
}

func InitializeErrResponder() {
	//initialize the counters
	counters = make(map[counterKey]int64)
	countersIn = make(chan counterKey, 64)
	go counterRoutine()
}

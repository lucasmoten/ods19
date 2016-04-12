package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"sync"

	"decipher.com/oduploader/util"
)

var (
	// The counters for error codes
	counters = make(map[counterKey]int64)
	// For this case, mutex is simpler than channels
	mutex = &sync.Mutex{}
)

// NewAppError constructs an application error
func NewAppError(code int, err error, msg string) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:  code,
		Error: err,
		Msg:   msg,
		File:  file,
		Line:  line,
	}
}

func countOKResponse() {
	sendErrorResponseRaw(nil, nil, 1)
}

// sendError lets us send the new AppError type before getting rid of server.AppError
func sendError(w *http.ResponseWriter, err error, msg string) {
	herr := &AppError{}
	switch e := err.(type) {
	case *util.AppError:
		//Copy over similar structure
		herr.File = e.File
		herr.Line = e.Line
		herr.Msg = msg
		// We guess the code based on the kind of uerr here
		//For now, treat a dependency like it's out bug.  It's not the user's bug
		if e.BlameBug || e.BlameDependency {
			herr.Code = 500
		}
		if e.BlameInput {
			herr.Code = 400
		}
		// util.AppError is really of type error so .Error() conflicts with .Error
		herr.Error = e.Err
		sendErrorResponseRaw(w, herr, 1)
	case util.AppError:
		//Copy over similar structure
		herr.File = e.File
		herr.Line = e.Line
		herr.Msg = msg
		// We guess the code based on the kind of uerr here
		if e.BlameBug {
			herr.Code = 500
		}
		// We guess the code based on the kind of uerr here
		//For now, treat a dependency like it's out bug.  It's not the user's bug
		if e.BlameBug || e.BlameDependency {
			herr.Code = 500
		}
		// util.AppError is really of type error so .Error() conflicts with .Error
		herr.Error = e.Err
		sendErrorResponseRaw(w, herr, 1)
	default:
		//If we are given no diagnostic information, then assume that 500 is the code
		_, file, line, _ := runtime.Caller(1)
		sendErrorResponseRaw(w, &AppError{500, err, msg, file, line}, 1)
	}
}

func sendErrorResponse(w *http.ResponseWriter, code int, err error, msg string) {
	_, file, line, _ := runtime.Caller(1)
	sendErrorResponseRaw(w, &AppError{code, err, msg, file, line}, 1)
}

func sendAppErrorResponse(w *http.ResponseWriter, herr *AppError) {
	sendErrorResponseRaw(w, herr, 1)
}

// sendErrorResponse is the publicly called function for sending error response from top level handlers
func sendErrorResponseRaw(w *http.ResponseWriter, herr *AppError, indirection int) {
	pc, _, _, _ := runtime.Caller(1 + indirection)
	endpointParts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	endpoint := endpointParts[len(endpointParts)-1]
	if herr != nil {
		log.Printf("httpCode %d:%s at %s:%d %s:%v", herr.Code, endpoint, herr.File, herr.Line, herr.Msg, herr.Error)
		mutex.Lock()
		counters[counterKey{herr.Code, endpoint, herr.File, herr.Line}]++
		mutex.Unlock()
		if w != nil {
			http.Error(*w, herr.Msg, herr.Code)
		}
	} else {
		//It's implicitly a 200
		mutex.Lock()
		counters[counterKey{200, endpoint, "", 0}]++
		mutex.Unlock()
	}
}

// writeCounters lets us write the counters out to stats
func renderErrorCounters(w http.ResponseWriter) {
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

// Write the counters out.  Make sure we are in the thread of the datastructure when we do this
func doWriteCounters(w http.ResponseWriter) {

	//Count the total number of events per endpoint, and report for each line
	// This call can stall the whole server while it does its print outs.
	endpointTotals := make(map[string]int64)
	var lines = make([]string, 0)

	//We are under the lock, so don't do IO in here yet.
	mutex.Lock()
	for k, v := range counters {
		endpointTotals[k.Endpoint] += v
	}
	for k, v := range counters {
		if k.Code != 200 {
			lines = append(
				lines,
				fmt.Sprintf("%d/%d %d %s %s:%d", v, endpointTotals[k.Endpoint], k.Code, k.Endpoint, k.File, k.Line),
			)
		}
	}
	mutex.Unlock()

	//Do io outside the mutex!
	fmt.Fprintf(w, "ratio code endpoint file:line\n")
	for i := range lines {
		fmt.Fprintf(w, "%s\n", lines[i])
	}
}

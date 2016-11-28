package server

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"github.com/uber-go/zap"
)

var (
	// The counters for error codes
	counters = make(map[counterKey]int64)
	// For this case, mutex is simpler than channels
	mutex = &sync.Mutex{}
)

// NewAppError constructs an application error
func NewAppError(code int, err error, msg string, fields ...zap.Field) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:   code,
		Error:  err,
		Msg:    msg,
		File:   file,
		Line:   line,
		Fields: fields,
	}
}

func countOKResponse(logger zap.Logger) {
	sendErrorResponseRaw(logger, nil, nil)
}

// sendError lets us send the new AppError type before getting rid of server.AppError
func sendError(logger zap.Logger, w *http.ResponseWriter, err error, msg string, fields ...zap.Field) {
	//If we are given no diagnostic information, then assume that 500 is the code
	_, file, line, _ := runtime.Caller(1)
	sendErrorResponseRaw(logger, w, &AppError{500, err, msg, file, line, fields})
}

func sendErrorResponse(logger zap.Logger, w *http.ResponseWriter, code int, err error, msg string, fields ...zap.Field) {
	_, file, line, _ := runtime.Caller(1)
	sendErrorResponseRaw(logger, w, &AppError{code, err, msg, file, line, fields})
}

func sendAppErrorResponse(logger zap.Logger, w *http.ResponseWriter, herr *AppError) {
	sendErrorResponseRaw(logger, w, herr)
}

//Some codes have already had to have been set because an http body follows
//It's mostly just 200 and 206 that have http bodies
func alreadySent(code int) bool {
	switch code {
	case http.StatusPartialContent, http.StatusOK:
		return true
	default:
		return false
	}
}

// sendErrorResponse is the publicly called function for sending error response from top level handlers
func sendErrorResponseRaw(logger zap.Logger, w *http.ResponseWriter, herr *AppError) {
	if herr != nil {
		var herrString string
		if herr.Error != nil {
			herrString = herr.Error.Error()
		}
		//Pre-append our fields to the field list
		var fields []zap.Field
		fields = append(fields, zap.Int("status", herr.Code))
		fields = append(fields, zap.String("message", herr.Msg))
		fields = append(fields, zap.String("err", herrString))
		fields = append(fields, zap.String("file", herr.File))
		fields = append(fields, zap.Int("line", herr.Line))
		for _, v := range herr.Fields {
			fields = append(fields, v)
		}
		if herr.Code < 400 {
			logger.Info("transaction end", fields...)
		} else {
			if herr.Code < 500 {
				logger.Warn("transaction end", fields...)
			} else {
				logger.Error("transaction end", fields...)
			}
		}
		mutex.Lock()
		counters[counterKey{herr.Code, herr.File, herr.Line}]++
		mutex.Unlock()
		if w != nil && !alreadySent(herr.Code) {
			http.Error(*w, herr.Msg, herr.Code)
		}
	} else {
		logger.Info("transaction end",
			zap.Int("status", 200),
		)
		//It's implicitly a 200 - or some other OK where we sent back a nil error
		mutex.Lock()
		counters[counterKey{200, "", 0}]++
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
	Code int
	//Endpoint string
	//file:line are not necessarily required, but they do help to isolate exactly which code location
	File string
	Line int
}

// Write the counters out.  Make sure we are in the thread of the datastructure when we do this
func doWriteCounters(w http.ResponseWriter) {

	//Count the total number of events per endpoint, and report for each line
	// This call can stall the whole server while it does its print outs.
	//endpointTotals := make(map[string]int64)
	totalQueries := int64(0)
	totalErrors := int64(0)
	var lines = make([]string, 0)

	//We are under the lock, so don't do IO in here yet.
	mutex.Lock()
	for _, v := range counters {
		totalQueries += v
	}
	for k, v := range counters {
		//Unless it's 400 or greater, it's not an error.
		if k.Code >= 400 {
			lines = append(
				lines,
				fmt.Sprintf("%d\t%d\t%s:%d", v, k.Code, k.File, k.Line),
			)
			totalErrors += v
		}
	}
	mutex.Unlock()

	//Do io outside the mutex!
	if len(lines) == 0 {
		fmt.Fprintf(w, "Errors: none\n")
	} else {
		fmt.Fprintf(w, "Errors: %d in %d queries\n", totalErrors, totalQueries)
		fmt.Fprintf(w, "count\tcode\tfile:line\n")
		for i := range lines {
			fmt.Fprintf(w, "%s\n", lines[i])
		}
	}
}

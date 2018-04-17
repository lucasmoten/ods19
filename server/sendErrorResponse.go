package server

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// The counters for error codes
	counters = make(map[counterKey]int64)
	// For this case, mutex is simpler than channels
	mutex = &sync.Mutex{}
)

// NewAppError constructs an application error
func NewAppError(code int, err error, msg string, fields ...zapcore.Field) *AppError {
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

func countOKResponse(logger *zap.Logger) {
	sendErrorResponseRaw(logger, nil, nil)
}

// sendError lets us send the new AppError type before getting rid of server.AppError
func sendError(logger *zap.Logger, w *http.ResponseWriter, err error, msg string, fields ...zapcore.Field) {
	//If we are given no diagnostic information, then assume that 500 is the code
	_, file, line, _ := runtime.Caller(1)
	sendErrorResponseRaw(logger, w, &AppError{http.StatusInternalServerError, err, msg, file, line, fields})
}

func sendErrorResponse(logger *zap.Logger, w *http.ResponseWriter, code int, err error, msg string, fields ...zapcore.Field) {
	_, file, line, _ := runtime.Caller(1)
	sendErrorResponseRaw(logger, w, &AppError{code, err, msg, file, line, fields})
}

func sendAppErrorResponse(logger *zap.Logger, w *http.ResponseWriter, herr *AppError) {
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
func sendErrorResponseRaw(logger *zap.Logger, w *http.ResponseWriter, herr *AppError) {
	if herr != nil {
		var herrString string
		if herr.Error != nil {
			herrString = herr.Error.Error()
		}
		//Pre-append our fields to the field list
		var fields []zapcore.Field
		fields = append(fields, zap.Int("status", herr.Code))
		fields = append(fields, zap.String("message", herr.Msg))
		fields = append(fields, zap.Error(fmt.Errorf(herrString)))
		fields = append(fields, zap.String("file", herr.File))
		fields = append(fields, zap.Int("line", herr.Line))
		for _, v := range herr.Fields {
			fields = append(fields, v)
		}
		if herr.Code < http.StatusBadRequest {
			//1xx-3xx series
			logger.Info("transaction finish", fields...)
		} else {
			if herr.Code != http.StatusInternalServerError {
				//4xx series (client error)
				logger.Info("transaction finish", fields...)
			} else {
				//5xx series (server error)
				logger.Error("transaction finish", fields...)
			}
		}
		mutex.Lock()
		counters[counterKey{herr.Code, herr.File, herr.Line}]++
		mutex.Unlock()
		if w != nil && !alreadySent(herr.Code) {
			http.Error(*w, herr.Msg, herr.Code)
		}
	} else {
		logger.Info("transaction finish",
			zap.Int("status", http.StatusOK),
		)
		//It's implicitly a 200 - or some other OK where we sent back a nil error
		mutex.Lock()
		counters[counterKey{http.StatusOK, "", 0}]++
		mutex.Unlock()
	}
}

// We key counters by code and endpoint tuple
type counterKey struct {
	Code int
	//Endpoint string
	//file:line are not necessarily required, but they do help to isolate exactly which code location
	File string
	Line int
}

package util

import (
	"runtime"
)

// AppError allows us to retain file:line information and to classify
// errors, and to set (a superset of) http status codes if necessary.
type AppError struct {
	//A file where this error is constructed
	File string
	//The line of code where this error is constructed
	Line int
	//The message to show the user
	Msg string
	//An error that caused this to be constructed
	// Don't show this to the user, as it may contain sensitive information.
	Err error
	//If this is the result of bad input
	BlameInput bool
	//If this is the result of a bug
	BlameBug bool
	//If this is the result of a bad dependency that we have no control over
	BlameDependency bool
	//Status code, a superset of http codes.  Ignore code zero.
	//If this is set, then the intention is to make this bubble all the way to the top of the endpoint.
	Code int
}

// Satisfy the error interface
func (e AppError) Error() string {
	return e.Msg
}

// NewAppError constructs an app error at this location
// Use Sprintf yourself if you need a variable string constructor
func NewAppError(code int, err error, msg string) *AppError {
	e := &AppError{
		Code: code,
		Msg:  msg,
		Err:  err,
	}
	//If we want to assign implicit blame, then use these ranges
	if 500 <= code && code <= 599 {
		e.BlameBug = true
	}
	if 400 <= code && code <= 499 {
		e.BlameInput = true
	}
	_, file, line, _ := runtime.Caller(1)
	e.File = file
	e.Line = line
	return e
}

// NewAppErrorBug constructs an app error at this location
func NewAppErrorBug(err error, msg string) *AppError {
	e := &AppError{
		Msg:      msg,
		Err:      err,
		BlameBug: true,
	}
	_, file, line, _ := runtime.Caller(1)
	e.File = file
	e.Line = line
	return e
}

// NewAppErrorInput constructs an app error at this location
func NewAppErrorInput(err error, msg string) *AppError {
	e := &AppError{
		Msg:        msg,
		Err:        err,
		BlameInput: true,
	}
	_, file, line, _ := runtime.Caller(1)
	e.File = file
	e.Line = line
	return e
}

// NewAppErrorDependency constructs an app error at this location
func NewAppErrorDependency(err error, msg string) *AppError {
	e := &AppError{
		Msg:             msg,
		Err:             err,
		BlameDependency: true,
	}
	_, file, line, _ := runtime.Caller(1)
	e.File = file
	e.Line = line
	return e
}

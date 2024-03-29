package util

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//
// Loggable gives us simple errors that are comparable by message - keep named args separate
// Use in place of error type:
//
//      err := someFunc()
//      err.ToInfo(logger)
//
//      // Optional named args, and supply an err to avoid err != nil before log checks everywhere
//      return NewLoggable("ciphertextcache cannot stat", err)
//
type Loggable struct {
	Msg  string
	Args []zapcore.Field
}

// Satisfy the error interface
func (e Loggable) Error() string {
	return e.Msg
}

// ToInfo writes to zap Info
func (e Loggable) ToInfo(logger *zap.Logger) {
	logger.Info(e.Msg, e.Args...)
}

// ToError writes to zap Error
func (e Loggable) ToError(logger *zap.Logger) {
	logger.Error(e.Msg, e.Args...)
}

// ToFatal writes to zap Fatal
func (e Loggable) ToFatal(logger *zap.Logger) {
	logger.Fatal(e.Msg, e.Args...)
}

// NewLoggable with vararg parameters
func NewLoggable(msg string, err error, args ...zapcore.Field) *Loggable {
	if err != nil {
		args = append(args, zap.Error(err))
	}
	return &Loggable{msg, args}
}

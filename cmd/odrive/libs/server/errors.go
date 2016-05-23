package server

import "github.com/uber-go/zap"

// AppError encapsulates an error with a desired http status code so that the server
// can issue the error code to the client.
// At points where a goroutine originating from a ServerHTTP call
// must stop and issue an error to the client and stop any further information in the
// connection.  This AppError is *not* recoverable in any way, because the connection
// is already considered dead at this point.  At best, intermediate handlers may need
// to handle surrounding cleanup that wasn't already done with a defer.
//
//  If we are not in the top level handler, we should always just stop and quietly throw
//  the error up:
//
//    if a,b,herr,err := h.acceptUpload(......); herr != nil {
//      return herr
//    }
//
//  And the top level ServeHTTP (or as high as possible) needs to handle it for real, and stop.
//
//     if herr != nil {
//         h.sendError(herr.Code, herr.Err, herr.Msg)
//         return //DO NOT RECOVER.  THE HTTP ERROR CODES HAVE BEEN SENT!
//     }
//
type AppError struct {
	Code   int         //the http error code to return with the msg
	Error  error       //an error that is ONLY for the log.  showing to the user may be sensitive.
	Msg    string      //message to show to the user, and in log
	File   string      //origin file
	Line   int         //origin line
	Fields []zap.Field //Set of arguments for the msg so that msg can be constant
}

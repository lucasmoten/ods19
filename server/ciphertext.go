package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"decipher.com/object-drive-server/ciphertext"
	"github.com/uber-go/zap"
	"golang.org/x/net/context"
)

//
// If a peer can't get ciphertext from PermanentStorage, then it can ask around to see who has it.
// If we get asked, we can serve it back to the caller.  If we don't ask peers, we can be
// stuck trying to get the ciphertext from PermanentStorage in a very long stall that will time out.
//
// Also if PermanentStorage is disabled with a load balanced setup, the ciphertext would not come back at all
// without p2p requesting.
//
func (h AppServer) getCiphertext(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	if r.Header.Get("USER_DN") != ciphertext.PeerSignifier {
		return NewAppError(403, fmt.Errorf("p2p required to get ciphertext"), "forbidden")
	}
	//We are getting a p2p ciphertext request, so that we can handle getting range requests
	//before a file can make it into PermanentStorage
	logger := LoggerFromContext(ctx)
	//Ask a drain provider directly to give us a particular ciphertext.
	captureGroups, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		return NewAppError(400, nil, "unparseable uri parameters")
	}

	//Specify which ciphertext out of which drain provider we are looking for
	selector := ciphertext.CiphertextCacheName(captureGroups["selector"])
	rName := ciphertext.FileId(captureGroups["rname"])
	dp := ciphertext.FindCiphertextCache(selector)

	//If there is a byte range, then use it.
	startAt := int64(0)
	byteRange, err := extractByteRange(r)
	if err != nil {
		return NewAppError(400, err, "byte range parse fail")
	}
	//We just want to know where to start from, and stream the whole file
	//until the client stops reading it.
	if byteRange != nil {
		startAt = byteRange.Start
	}

	//Send back the byte range asked for
	f, length, err := ciphertext.UseLocalFile(logger, dp, rName, startAt)
	if err != nil {
		//Keep it quiet in the case of not found
		return NewAppError(500, err, "error looking in p2p cache")
	}
	if f == nil {
		return NewAppError(204, nil, "not in this p2p cache")
	}
	if length < 0 {
		logger.Error("p2p bad length", zap.Int64("Content-Length", length))
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	//w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
	byteCount, err := io.Copy(w, f)

	//It is perfectly normal for a client to only pull part of the data and cut us off
	if err != nil && strings.Contains(err.Error(), "write: connection reset by peer") == false {
		logger.Info("p2p copy failure", zap.String("err", err.Error()), zap.Int64("bytes", byteCount))
	}
	return nil
}
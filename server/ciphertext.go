package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/services/audit"
	"go.uber.org/zap"
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
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")
	if r.Header.Get("USER_DN") != ciphertext.PeerSignifier {
		herr := NewAppError(http.StatusForbidden, fmt.Errorf("p2p required to get ciphertext"), "forbidden")
		h.publishError(gem, herr)
		return herr
	}
	//We are getting a p2p ciphertext request, so that we can handle getting range requests
	//before a file can make it into PermanentStorage
	logger := LoggerFromContext(ctx)
	//Ask a drain provider directly to give us a particular ciphertext.
	captureGroups, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		herr := NewAppError(http.StatusBadRequest, nil, "unparseable uri parameters")
		h.publishError(gem, herr)
		return herr
	}

	//Specify which ciphertext out of which drain provider we are looking for
	zone := ciphertext.CiphertextCacheZone(captureGroups["zone"])
	rName := ciphertext.FileId(captureGroups["rname"])
	dp := ciphertext.FindCiphertextCache(zone)

	//If there is a byte range, then use it.
	startAt := int64(0)
	byteRange, err := extractByteRange(r)
	if err != nil {
		herr := NewAppError(http.StatusBadRequest, err, "byte range parse fail")
		h.publishError(gem, herr)
		return herr
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
		herr := NewAppError(http.StatusInternalServerError, err, "error looking in p2p cache")
		h.publishError(gem, herr)
		return herr
	}
	if f == nil {
		herr := NewAppError(http.StatusNoContent, nil, "not in this p2p cache")
		h.publishSuccess(gem, w)
		return herr
	}
	if length < 0 {
		herr := NewAppError(http.StatusInternalServerError, nil, "p2p bad legnth")
		h.publishError(gem, herr)
		logger.Error("p2p bad length", zap.Int64("Content-Length", length))
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	//w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
	byteCount, err := io.Copy(w, f)

	//It is perfectly normal for a client to only pull part of the data and cut us off
	if err != nil && strings.Contains(err.Error(), "write: connection reset by peer") == false {
		logger.Info("p2p copy failure", zap.Error(err), zap.Int64("bytes", byteCount))
	}
	h.publishSuccess(gem, w)
	return nil
}

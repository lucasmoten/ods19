package server

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/oduploader/performance"
)

func (h AppServer) getStats(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\nErrors:\n")
	renderErrorCounters(w)

	fmt.Fprintf(w, "\nUploaders Aggregate:\n")
	h.Tracker.Reporters[performance.UploadCounter].Q.Dump(w)

	fmt.Fprintf(w, "\nDownloaders Aggregate:\n")
	h.Tracker.Reporters[performance.DownloadCounter].Q.Dump(w)

	fmt.Fprintf(w, "\nToS3:\n")
	h.Tracker.Reporters[performance.S3DrainTo].Q.Dump(w)

	fmt.Fprintf(w, "\nFrom S3:\n")
	h.Tracker.Reporters[performance.S3DrainFrom].Q.Dump(w)

	fmt.Fprintf(w, "\nFrom AAC:\n")
	h.Tracker.Reporters[performance.AACCounter].Q.Dump(w)

	countOKResponse()
}

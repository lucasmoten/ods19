package server

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/performance"
)

func (h AppServer) getStats(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\nErrors:\n")
	renderErrorCounters(w)

	verboseParameter := r.URL.Query().Get("verbose")
	verbose := false
	if verboseParameter == "true" {
		verbose = true
	}

	fmt.Fprintf(w, "\nUploaders Aggregate:\n")
	h.Tracker.Reporters[performance.UploadCounter].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nDownloaders Aggregate:\n")
	h.Tracker.Reporters[performance.DownloadCounter].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nToS3:\n")
	h.Tracker.Reporters[performance.S3DrainTo].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nFrom S3:\n")
	h.Tracker.Reporters[performance.S3DrainFrom].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nFrom AAC:\n")
	h.Tracker.Reporters[performance.AACCounter].Q.Dump(w, verbose)

	countOKResponse()
}

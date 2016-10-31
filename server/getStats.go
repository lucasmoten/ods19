package server

import (
	"fmt"
	"net/http"

	"decipher.com/object-drive-server/ciphertext"
	"golang.org/x/net/context"

	"decipher.com/object-drive-server/autoscale"
	"decipher.com/object-drive-server/performance"
)

func (h AppServer) getStats(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	renderErrorCounters(w)

	verboseParameter := r.URL.Query().Get("verbose")
	verbose := false
	if verboseParameter == "true" {
		verbose = true
	}

	fmt.Fprintf(w, "\nLast Cloudwatch report\n")
	autoscale.CloudWatchDump(w)

	fmt.Fprintf(w, "\nUploaders Aggregate:\n")
	h.Tracker.Reporters[performance.UploadCounter].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nDownloaders Aggregate:\n")
	h.Tracker.Reporters[performance.DownloadCounter].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nToS3:\n")
	h.Tracker.Reporters[performance.S3DrainTo].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nFrom S3:\n")
	h.Tracker.Reporters[performance.S3DrainFrom].Q.Dump(w, verbose)

	fmt.Fprintf(w, "\nFrom AAC:\n")
	fmt.Fprintf(w, "\n- Check Access:\n")
	h.Tracker.Reporters[performance.AACCounterCheckAccess].Q.Dump(w, verbose)
	fmt.Fprintf(w, "\n- Get Snippets:\n")
	h.Tracker.Reporters[performance.AACCounterGetSnippets].Q.Dump(w, verbose)

	caches := ciphertext.FindCiphertextCacheList()
	for dpName, dp := range caches {
		fmt.Fprintf(w, "\nCiphertextCache %s:\n", dpName)
		dp.CacheInventory(w, verbose)
	}

	return nil
}

package server

import (
	"fmt"
	"net/http"

	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/services/audit"
	"golang.org/x/net/context"

	"decipher.com/object-drive-server/autoscale"
	"decipher.com/object-drive-server/performance"
)

func (h AppServer) getStats(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

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
		fmt.Fprintf(w, "\nCiphertextCache %d:\n", dpName)
		dp.CacheInventory(w, verbose)
	}
	h.publishSuccess(gem, w)
	return nil
}

// writeCounters lets us write the counters out to stats
func renderErrorCounters(w http.ResponseWriter) {
	doWriteCounters(w)
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

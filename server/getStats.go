package server

import (
	"fmt"
	"net/http"
	"time"

	metrics "github.com/rcrowley/go-metrics"

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

	fmt.Fprintf(w, "\n\n")
	// Log go-metrics
	scale := time.Millisecond
	du := float64(scale)
	duSuffix := scale.String()[1:]
	metrics.DefaultRegistry.Each(
		func(name string, i interface{}) {
			metric, ok := i.(metrics.Timer)
			if ok {
				t := metric.Snapshot()
				if t.Count() > 0 {
					ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
					fmt.Fprintf(w, "timer %s\n", name)
					fmt.Fprintf(w, "  count:       %9d\n", t.Count())
					fmt.Fprintf(w, "  min:         %12.2f%s\n", float64(t.Min())/du, duSuffix)
					fmt.Fprintf(w, "  max:         %12.2f%s\n", float64(t.Max())/du, duSuffix)
					fmt.Fprintf(w, "  mean:        %12.2f%s\n", t.Mean()/du, duSuffix)
					fmt.Fprintf(w, "  stddev:      %12.2f%s\n", t.StdDev()/du, duSuffix)
					fmt.Fprintf(w, "  median:      %12.2f%s\n", ps[0]/du, duSuffix)
					fmt.Fprintf(w, "  75%%:         %12.2f%s\n", ps[1]/du, duSuffix)
					fmt.Fprintf(w, "  95%%:         %12.2f%s\n", ps[2]/du, duSuffix)
					fmt.Fprintf(w, "  99%%:         %12.2f%s\n", ps[3]/du, duSuffix)
					fmt.Fprintf(w, "  99.9%%:       %12.2f%s\n", ps[4]/du, duSuffix)
					fmt.Fprintf(w, "  1-min rate:  %12.2f\n", t.Rate1())
					fmt.Fprintf(w, "  5-min rate:  %12.2f\n", t.Rate5())
					fmt.Fprintf(w, "  15-min rate: %12.2f\n", t.Rate15())
					fmt.Fprintf(w, "  mean rate:   %12.2f\n", t.RateMean())
				}
			}
		},
	)
}

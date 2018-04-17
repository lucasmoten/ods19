package server

import (
	"fmt"
	"net/http"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"github.com/deciphernow/object-drive-server/services/audit"
	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/autoscale"
)

func (h AppServer) getStats(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")
	// Explicitely set content-type otherwise browser will think this is plain/text
	w.Header().Add("content-type", "application/json")

	fmt.Fprint(w, "{\n")

	autoscale.CloudWatchDump(w)
	renderErrorCounters(w)

	fmt.Fprint(w, "}\n")

	h.publishSuccess(gem, w)
	return nil
}

// Write the counters out.  Make sure we are in the thread of the datastructure when we do this
func renderErrorCounters(w http.ResponseWriter) {
	// Count the total number of events per endpoint, and report for each line
	// This call can stall the whole server while it does its print outs.
	totalQueries := int64(0)
	totalErrors := int64(0)

	// We are under the lock, so don't do IO in here yet.
	mutex.Lock()
	for _, v := range counters {
		totalQueries += v
	}
	for k, v := range counters {
		// Unless it's 400 or greater, it's not an error.
		if k.Code >= http.StatusBadRequest {
			totalErrors += v
		}
	}
	mutex.Unlock()

	// Log go-metrics
	scale := time.Millisecond
	du := float64(scale)
	metrics.DefaultRegistry.Each(
		func(name string, i interface{}) {
			metric, ok := i.(metrics.Timer)
			if ok {
				t := metric.Snapshot()
				if t.Count() > 0 {
					ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
					fmt.Fprintf(w, "\t\"%s/count\": %9d,\n", name, t.Count())
					fmt.Fprintf(w, "\t\"%s/latency_ms.min\": %12.2f,\n", name, float64(t.Min())/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.max\": %12.2f,\n", name, float64(t.Max())/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.mean\": %12.2f,\n", name, float64(t.Mean())/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.stddev\": %12.2f,\n", name, t.StdDev()/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.p50\": %12.2f,\n", name, ps[0]/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.p75\": %12.2f,\n", name, ps[1]/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.p95\": %12.2f,\n", name, ps[2]/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.p99\": %12.2f,\n", name, ps[3]/du)
					fmt.Fprintf(w, "\t\"%s/latency_ms.p999\": %12.2f,\n", name, ps[4]/du)
					fmt.Fprintf(w, "\t\"%s/1_min_rate\": %12.2f,\n", name, t.Rate1())
					fmt.Fprintf(w, "\t\"%s/5_min_rate\": %12.2f,\n", name, t.Rate5())
					fmt.Fprintf(w, "\t\"%s/15_min_rate\": %12.2f,\n", name, t.Rate15())
					fmt.Fprintf(w, "\t\"%s/mean_rate\": %12.2f,\n", name, t.RateMean())
				}
			}
		},
	)

	//Move this to the end so that we know that the last entry has no trailing comma
	fmt.Fprintf(w, "\t\"odrive/total_errors/\": %d,\n", totalErrors)
	fmt.Fprintf(w, "\t\"odrive/total_queries/\": %d\n", totalQueries)
}

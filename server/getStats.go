package server

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/services/audit"
	"golang.org/x/net/context"

	"bitbucket.di2e.net/dime/object-drive-server/autoscale"
)

func (h AppServer) getStats(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")
	// Explicitly set content-type otherwise browser will think this is plain/text
	w.Header().Add("content-type", "application/json")

	// Open
	fmt.Fprint(w, "{\n")

	// Our statistics to report
	t := time.Now().UTC()
	fmt.Fprintf(w, "\t\"statsReportedDate\": \"%s\",\n", t.Format(time.RFC3339Nano))
	hostname, _ := os.Hostname()
	fmt.Fprintf(w, "\t\"nodeId\": \"%s\",\n", config.NodeID)
	fmt.Fprintf(w, "\t\"hostname\": \"%s\",\n", hostname)
	fmt.Fprintf(w, "\t\"databaseConnectionCount\": %d,\n", h.RootDAO.GetOpenConnectionCount())
	autoscale.CloudWatchDump(w) // includes CPU utilization, etc
	fmt.Fprintf(w, "\t\"usersLruCacheCount\": %d,\n", h.UsersLruCache.ItemCount())
	fmt.Fprintf(w, "\t\"userAOsLruCacheCount\": %d,\n", h.UserAOsLruCache.ItemCount())
	fmt.Fprintf(w, "\t\"typesLruCacheCount\": %d,\n", h.TypeLruCache.ItemCount())
	renderErrorCounters(w)
	renderMetricsForTrackedFunctions(w)

	// Close
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
	userErrors := int64(0)
	serverErrors := int64(0)

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
		if k.Code >= http.StatusBadRequest && k.Code < http.StatusInternalServerError {
			userErrors += v
		}
		if k.Code >= http.StatusInternalServerError {
			serverErrors += v
		}
	}
	mutex.Unlock()
	fmt.Fprintf(w, "\t\"trackedRequests\": {\n")
	fmt.Fprintf(w, "\t\t\"queryCount\": %d,\n", totalQueries)
	fmt.Fprintf(w, "\t\t\"errorCount\": %d,\n", totalErrors)
	fmt.Fprintf(w, "\t\t\"userErrorCount\": %d,\n", userErrors)
	fmt.Fprintf(w, "\t\t\"serverErrorCount\": %d\n", serverErrors)
	fmt.Fprintf(w, "\t},\n")
}

func renderMetricsForTrackedFunctions(w http.ResponseWriter) {

	var metrickeys []string
	metrics.DefaultRegistry.Each(
		func(name string, i interface{}) {
			metrickeys = append(metrickeys, name)
		},
	)
	sort.Strings(metrickeys)

	// Log go-metrics
	scale := time.Millisecond
	du := float64(scale)
	fmt.Fprintf(w, "\t\"%s\": {\n", "trackedFunctions")
	kl := len(metrickeys)
	for ki, k := range metrickeys {
		name := k
		normalizedName := sanitizeMyName(name)
		i := metrics.DefaultRegistry.Get(k)
		metric, ok := i.(metrics.Timer)
		if ok {
			t := metric.Snapshot()
			if t.Count() > 0 {
				fmt.Fprintf(w, "\t\t\"%s\": {\n", normalizedName)
				fmt.Fprintf(w, "\t\t\t\"givenname\": \"%s\",\n", name)
				ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				fmt.Fprintf(w, "\t\t\t\"count\": %9d,\n", t.Count())
				fmt.Fprintf(w, "\t\t\t\"latency_ms.min\": %12.2f,\n", float64(t.Min())/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.max\": %12.2f,\n", float64(t.Max())/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.mean\": %12.2f,\n", float64(t.Mean())/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.stddev\": %12.2f,\n", t.StdDev()/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.p50\": %12.2f,\n", ps[0]/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.p75\": %12.2f,\n", ps[1]/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.p95\": %12.2f,\n", ps[2]/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.p99\": %12.2f,\n", ps[3]/du)
				fmt.Fprintf(w, "\t\t\t\"latency_ms.p999\": %12.2f,\n", ps[4]/du)
				fmt.Fprintf(w, "\t\t\t\"1_min_rate\": %12.2f,\n", t.Rate1())
				fmt.Fprintf(w, "\t\t\t\"5_min_rate\": %12.2f,\n", t.Rate5())
				fmt.Fprintf(w, "\t\t\t\"15_min_rate\": %12.2f,\n", t.Rate15())
				fmt.Fprintf(w, "\t\t\t\"mean_rate\": %12.2f\n", t.RateMean())
				if ki < kl-1 {
					fmt.Fprintf(w, "\t\t},\n")
				} else {
					fmt.Fprintf(w, "\t\t}\n")
				}
			}
		}
	}
	fmt.Fprintf(w, "\t}\n")
}

func sanitizeMyName(inVal string) string {
	reg, _ := regexp.Compile("[^A-Za-z0-9]+")
	return strings.ToLower(reg.ReplaceAllString(inVal, "_"))
}

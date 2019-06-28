package autoscale

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/amazon"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/performance"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	linuxproc "github.com/c9s/goprocinfo/linux"
	"go.uber.org/zap"
)

var (
	cwAccumulatorMutex = &sync.Mutex{}
	cwAccumulator      = &CloudWatchStatsAccumulator{}
	cwStats            = &CloudWatchGeneralStats{}
)

// CloudWatchStatsAccumulator is where we accumulate stats for this interval
type CloudWatchStatsAccumulator struct {
	RecentRequestTime  int64
	RecentRequestCount int64
	RecentSince        int64
	RecentBytes        int64
}

// CloudWatchGeneralStats is the data that we send up to CloudWatch
type CloudWatchGeneralStats struct {
	Latency         *float64
	Throughput      *float64
	CPUUtilization  *float64
	MemKB           *float64
	MemPct          *float64
	Load            *float64
	IntervalMillis  *int64
	IntervalLast    *int64
	IntervalNext    *int64
	SysMem          *float64
	FileDescriptors *int64
	GoRoutines      *int64
}

//CloudWatchDump shows the latest thing sent to CloudWatch
func CloudWatchDump(w io.Writer) {
	//Make sure we get a consistent copy so we don't get a pointer nulled in between check and use time
	cwAccumulatorMutex.Lock()
	renderedStats := *cwStats
	cwAccumulatorMutex.Unlock()
	//We do NOT want to do periodic logging of successful CloudWatch sends, or it will fill the logs with noise and make them large when nothing interesting is happening.
	//So, let us look at such stats here.
	if renderedStats.IntervalMillis != nil {
		fmt.Fprintf(w, "\t\"statssnapshot/interval_in_seconds\": %f,\n", float64(float64(*renderedStats.IntervalMillis)/float64(1000)))
		timeLastMSDuration := util.NowMS() - *renderedStats.IntervalLast
		fmt.Fprintf(w, "\t\"statssnapshot/interval_last\": \"%s\",\n", time.Now().Add(time.Duration(-1*timeLastMSDuration)*time.Millisecond).UTC().Format(time.RFC3339Nano))
		timeNextMSDuration := *renderedStats.IntervalNext - util.NowMS()
		fmt.Fprintf(w, "\t\"statssnapshot/interval_next\": \"%s\",\n", time.Now().Add(time.Duration(timeNextMSDuration)*time.Millisecond).UTC().Format(time.RFC3339Nano))
	}
	if renderedStats.CPUUtilization != nil {
		if !math.IsNaN(*renderedStats.CPUUtilization) {
			fmt.Fprintf(w, "\t\"statssnapshot/container/cpu_utilization_pct\": %f,\n", *renderedStats.CPUUtilization)
		}
	}
	if renderedStats.Load != nil {
		fmt.Fprintf(w, "\t\"statssnapshot/container/cpu_5min_loadavg\": %f,\n", *renderedStats.Load)
		fmt.Fprintf(w, "\t\"statssnapshot/container/cpu_count\": %d,\n", runtime.NumCPU())
	}
	if renderedStats.MemKB != nil {
		fmt.Fprintf(w, "\t\"statssnapshot/process/mem_heap_used_kb\": %9.f,\n", *renderedStats.MemKB)
	}
	if renderedStats.SysMem != nil {
		fmt.Fprintf(w, "\t\"statssnapshot/process/mem_from_os_kb\": %9.f,\n", *renderedStats.SysMem)
	}
	if renderedStats.FileDescriptors != nil {
		fmt.Fprintf(w, "\t\"statssnapshot/process/file_descriptors_count\": %d,\n", *renderedStats.FileDescriptors)
	}
	if renderedStats.GoRoutines != nil {
		fmt.Fprintf(w, "\t\"statssnapshot/process/go_routines_count\": %d,\n", *renderedStats.GoRoutines)
	}
	// 20190402 - Commenting these out because its way wrong in implementation, assuming success and instant transfers
	// if renderedStats.Throughput != nil {
	// 	fmt.Fprintf(w, "\t\"statssnapshot/process/throughput_kb\": %f,\n", *renderedStats.Throughput)
	// }
	// if renderedStats.Latency != nil {
	// 	fmt.Fprintf(w, "\t\"statssnapshot/process/latency_ms\": %f,\n", *renderedStats.Latency)
	// }
}

// CloudWatchTransaction wraps CloudWatchTransactionRaw with start/stop and bytes
func CloudWatchTransaction(start, stop int64, tracker *performance.JobReporters) {
	bytes := tracker.GetUploadDownloadByteTotal()
	CloudWatchTransactionRaw(start, stop, bytes)
}

// CloudWatchTransactionRaw lets a transaction input be marked directly
func CloudWatchTransactionRaw(start, stop, bytes int64) {
	//Just use total of upload and download bytes for now.  We don't have good metrics elsewhere
	//This is called at the end of every http request, so keep this simple so that
	//these never queue up.
	cwAccumulatorMutex.Lock()
	cwAccumulator.RecentRequestTime += (stop - start)
	cwAccumulator.RecentBytes += bytes
	cwAccumulator.RecentRequestCount++
	cwAccumulatorMutex.Unlock()
}

// CloudWatchStartInterval at the beginning of a cloudwatch sampling interval, invoke this
func CloudWatchStartInterval(tracker *performance.JobReporters, now int64) {
	cwAccumulatorMutex.Lock()
	tracker.NewInterval()
	cwAccumulator.RecentRequestCount = 0
	cwAccumulator.RecentRequestTime = 0
	cwAccumulator.RecentBytes = 0
	cwAccumulator.RecentSince = now
	cwAccumulatorMutex.Unlock()
}

// log debug info to cloudwatch that you can see if you set log level to debug
func logMetricDatum(logger *zap.Logger, d *cloudwatch.MetricDatum) {
	if d.Value == nil {
		return
	}
	logger.Debug(
		"cloudwatch datum",
		zap.String("MetricName", *d.MetricName),
		zap.Any("Dimensions", d.Dimensions),
		zap.String("Timestamp", fmt.Sprintf("%v", *d.Timestamp)),
		zap.String("Unit", *d.Unit),
		zap.Float64("Value", *d.Value),
	)
}

// GetProcStat gives us info required to compute cpu utilization related stats
func GetProcStat(logger *zap.Logger) *linuxproc.Stat {
	//You can only compute a cpu percentage relative to a previous reading.  So this is the one that starts the interval.
	var err error
	prevStat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		logger.Warn("stat read fail", zap.Error(err))
	}
	return prevStat
}

// LoadAvgStat is a parse of /proc/loadavg
type LoadAvgStat struct {
	CPU1Min          float64
	CPU5Min          float64
	CPU10Min         float64
	RunningProcesses int
	TotalProcesses   int
	LastPid          int
}

// GetLoadAvgStat gives us the info required to compute load average (same as top)
func GetLoadAvgStat(logger *zap.Logger) *LoadAvgStat {
	var err error
	f, err := os.Open("/proc/loadavg")
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		logger.Warn("loadavg fail to open", zap.Error(err))
		return nil
	}

	buffer := make([]byte, 1024)
	count, err := f.Read(buffer)
	if err != nil {
		logger.Warn("loadavg fail to parse", zap.Error(err))
		return nil
	}
	bufferString := string(buffer[:count])
	logger.Debug("stats interval parsing loadavg", zap.String("raw", bufferString))
	tokens := strings.Split(bufferString, " ")

	returnValue := &LoadAvgStat{}
	returnValue.CPU1Min, err = strconv.ParseFloat(tokens[0], 32)
	returnValue.CPU5Min, err = strconv.ParseFloat(tokens[1], 32)
	returnValue.CPU10Min, err = strconv.ParseFloat(tokens[2], 32)
	procRatio := strings.Split(tokens[3], "/")
	returnValue.RunningProcesses, err = strconv.Atoi(procRatio[0])
	returnValue.TotalProcesses, err = strconv.Atoi(procRatio[1])
	returnValue.LastPid, err = strconv.Atoi(tokens[4])

	return returnValue
}

func cpuTimeFromStat(s *linuxproc.Stat, i int) int64 {
	return (int64(s.CPUStats[i].User) - int64(s.CPUStats[i].Guest)) +
		(int64(s.CPUStats[i].Nice) - int64(s.CPUStats[i].GuestNice)) +
		int64(s.CPUStats[i].System) +
		int64(s.CPUStats[i].IRQ+s.CPUStats[i].SoftIRQ)
}

func idleTimeFromStat(s *linuxproc.Stat, i int) int64 {
	return int64(s.CPUStats[i].Idle) //+ int64(s.CPUStats[i].IOWait) // 20190402 removed IOWait as its a subset of Idle
}

// Interpret the parsed output of /proc/stat
func computeUtilization(prevStat, nextStat *linuxproc.Stat) (int64, int64) {
	//We are calculating utilization by simply adding up all reported CPU time (presuming that user+nice+system+idle+iowait+idle are disjoint counts),
	//and then subtracting idle time to get utilized time.   That means that overall, it's just non-idle over total time:  (Ttime - Itime)/Ttime
	//
	//NOTE: /proc/stat numbers are reported in *wierd* units.  Try to only take ratios so that you are not dealing with USER_HZ or Jiffies values,
	//which are not guaranteed to be a consistent unit across machines!
	//
	//	Vangelis Tasoulas answer: http://stackoverflow.com/questions/23367857/accurate-calculation-of-cpu-usage-given-in-percentage-in-linux
	//
	var totalCPUtime int64
	var totalWaitTime int64

	//Per CPU, total jiffies are spread per processor across: user,nice,system,idle,iowait,irq,softirq,steal,guest,guest_nice
	for i := range nextStat.CPUStats {
		//Compute an idle diff
		prevIdleTime := idleTimeFromStat(prevStat, i)
		idleTime := idleTimeFromStat(nextStat, i)
		//Include it in total wait time across CPUs
		totalWaitTime += (idleTime - prevIdleTime)
		//Compute a cpu diff
		prevCPUTime := cpuTimeFromStat(prevStat, i)
		cpuTime := cpuTimeFromStat(nextStat, i)
		//Include it in total time in the CPU
		totalCPUtime += (cpuTime - prevCPUTime)
	}
	return totalCPUtime, totalWaitTime
}

// ComputeOverallPerformance computes the numbers and remember stat state, and don't write into cloudwatch yet
func ComputeOverallPerformance(
	prevStat *linuxproc.Stat,
	nextStat *linuxproc.Stat,
	loadStat *LoadAvgStat,
	now int64,
) (*CloudWatchGeneralStats, *linuxproc.Stat) {
	//Get stats about memory
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	cwAccumulatorMutex.Lock()
	//These are the main inputs to in-server calculations
	millis := now - cwAccumulator.RecentSince
	bytes := cwAccumulator.RecentBytes
	requests := cwAccumulator.RecentRequestCount

	if millis > 0 {
		cwStats.IntervalMillis = aws.Int64(millis)
		cwStats.IntervalLast = aws.Int64(util.NowMS())
		cwStats.IntervalNext = aws.Int64(*cwStats.IntervalLast + millis)
		cwStats.Throughput = aws.Float64(float64(bytes) / float64(millis))
	}

	//There needs to be at least one request to have a latency value to report.
	//In this case, total ms of the observation interval divided by total number of requests is sufficient.
	if requests > 0 {
		cwStats.Latency = aws.Float64(float64(millis) / float64(requests))
	}

	cwStats.MemKB = aws.Float64(float64(mem.Alloc) / 1024)
	cwStats.MemPct = aws.Float64(100.0 * float64(mem.Alloc) / float64(mem.TotalAlloc))
	cwStats.SysMem = aws.Float64(float64(mem.Sys) / 1024)

	//Just take the 5min value from proc (note that we *also* have a 5min polling interval by default)
	cwStats.Load = aws.Float64(loadStat.CPU5Min)

	//Compute utilization of the CPUs
	totalCPUtime, totalWaitTime := computeUtilization(prevStat, nextStat)
	cwStats.CPUUtilization = aws.Float64(100.0 * float64(totalCPUtime) / float64(totalCPUtime+totalWaitTime))

	cwStats.FileDescriptors = aws.Int64(GetFileDescriptorCount())
	cwStats.GoRoutines = aws.Int64(int64(runtime.NumGoroutine()))

	//Just drop these numbers where the stats page can get to them so that we don't flood the Info logs with periodic cloudwatch success stats
	//during total inactivity.  You can accept the flood if you set to Debug though.
	cwAccumulatorMutex.Unlock()
	return cwStats, nextStat
}

// GetFileDescriptorCount gets the number of file descriptors associated with this process
func GetFileDescriptorCount() int64 {
	f, err := ioutil.ReadDir("/proc/self/fd")
	if err != nil {
		log.Printf("ERROR reading directory for count of file descriptors\n")
		log.Printf(err.Error())
		return -1
	}
	return int64(len(f))
}

// CloudWatchReportingStart begins the goroutine that publishes into CloudWatch
func CloudWatchReportingStart(tracker *performance.JobReporters) {
	//Get a session in which to work in a goroutine
	logger := config.RootLogger.With(zap.String("session", "cloudwatch"))

	//Try to get a real cloudwatch session.  If not, just log this data locally if enabled
	cwConfig := config.NewCWConfig()
	if cwConfig.SleepTimeInSeconds <= 0 {
		logger.Info("metrics reporting to cloudwatch disabled as OD_AWS_CLOUDWATCH_INTERVAL set to <= 0")
		// But compute performance to be able to report in stats
		go func() {
			prevStat := GetProcStat(logger)
			for {
				CloudWatchStartInterval(tracker, util.NowMS())
				time.Sleep(time.Duration(30) * time.Second)
				_, prevStat = ComputeOverallPerformance(prevStat, GetProcStat(logger), GetLoadAvgStat(logger), util.NowMS())
			}
		}()
		return
	}
	var cwSession *cloudwatch.CloudWatch
	var namespace *string

	if len(cwConfig.Name) == 0 {
		namespace = aws.String("nullCloudwatch")
	} else {
		//We use an immutable dimension that marks this as the odrive service, where we actually report to CloudWatch
		//for the IP (presuming they are unique, which is generally true outside of docker deployments)
		namespace = aws.String(cwConfig.Name)
		cwSession = cloudwatch.New(amazon.NewAWSSession(cwConfig.AWSConfig, logger, "cloudwatch"))
		if cwSession == nil {
			logger.Warn("cloudwatch txn fail on null session")
		}
		logger.Info("cloudwatch monitoring started", zap.String("implementation", *namespace))
	}

	var dims []*cloudwatch.Dimension
	dims = append(dims, &cloudwatch.Dimension{Name: aws.String("Service Name"), Value: aws.String("odrive")})

	//Just run in the background sending stats as we have them
	go func() {
		prevStat := GetProcStat(logger)
		for {
			CloudWatchStartInterval(tracker, util.NowMS())
			logger.Debug("cloudwatch wait", zap.Int("timeInSeconds", cwConfig.SleepTimeInSeconds))
			time.Sleep(time.Duration(cwConfig.SleepTimeInSeconds) * time.Second)
			logger.Debug("cloudwatch to report")

			//Get all the fields that we want to report from here
			_, prevStat = ComputeOverallPerformance(prevStat, GetProcStat(logger), GetLoadAvgStat(logger), util.NowMS())

			//Report metrics in the format that our cloudwatch setup is expecting

			var metricDatum []*cloudwatch.MetricDatum
			now := aws.Time(time.Now().UTC())
			if cwStats.Latency != nil {
				//Note: this is *not* 90th percentile latency.  It's just a simple latency over the last 5min interval.
				//We are computing this latency using these (with units):
				//  totalContentLength Bytes, totalTime Milliseconds, totalRequests None
				//
				//Note: because we include file uploads and downloads, this is NOT latency per byte.
				//High latency therefore does NOT necessarily indicate a problem.
				//It is often thought of as a problem when thinking about FIXED SIZED requests,
				//or due to its general correlation with load (ie: queueing in the system).
				//
				//Inverse of throughput is the service time for a kilobyte
				//So, You may want to use a low threshold on throughput to trigger an alarm - not latency.
				//ie:    Seconds/Kilobyte
				metricDatum = append(metricDatum,
					&cloudwatch.MetricDatum{
						MetricName: aws.String("srv/request_latency_ms.p90"),
						Dimensions: dims,
						Timestamp:  now,
						Unit:       aws.String("Milliseconds"),
						Value:      cwStats.Latency,
					},
				)
			}
			//This is based on a combination of upload and download AND idle time.
			if cwStats.Throughput != nil {
				metricDatum = append(metricDatum,
					&cloudwatch.MetricDatum{
						MetricName: aws.String("srv/throughput"),
						Dimensions: dims,
						Timestamp:  now,
						Unit:       aws.String("Kilobytes/Second"),
						Value:      cwStats.Throughput,
					},
				)
			}
			//This is the numbers for the whole container/vm/machine, because that's the unit that gets restarted
			if cwStats.CPUUtilization != nil {
				metricDatum = append(metricDatum,
					&cloudwatch.MetricDatum{
						MetricName: aws.String("process/cpu/percent"),
						Dimensions: dims,
						Timestamp:  now,
						Unit:       aws.String("Percent"), //actually, pct is a (scalar) unit applied to a ratio. none might be right by the api though.
						Value:      cwStats.CPUUtilization,
					},
				)
			}
			if cwStats.MemKB != nil {
				metricDatum = append(metricDatum,
					&cloudwatch.MetricDatum{
						MetricName: aws.String("process/memory/kb"),
						Dimensions: dims,
						Timestamp:  now,
						Unit:       aws.String("Kilobytes"), //actually, pct is a (scalar) unit applied to a ratio. none might be right by the api though.
						Value:      cwStats.MemKB,
					},
				)
			}
			//This is pct within the Go process
			if cwStats.MemPct != nil {
				metricDatum = append(metricDatum,
					&cloudwatch.MetricDatum{
						MetricName: aws.String("process/memory/percent"),
						Dimensions: dims,
						Timestamp:  now,
						Unit:       aws.String("Percent"), //actually, pct is a (scalar) unit applied to a ratio. none might be right by the api though.
						Value:      cwStats.MemPct,
					},
				)
			}
			//This is LoadAverage as reported by the OS
			//
			//This is probably an *excellent* metric to use for scaling decisions, since this
			//is the thing that we are trying to contain.  Very high load can cause latency to become
			//uncontrollably high, which is an indication of uncontrollably low throughput.
			//
			//But: note that you can have zero throughput because you have an idle system!
			// you can have high latency just because you are transferring large files
			// (ie: taking time without accounting for size).  If you account for size, then you
			// just get inverse of throughput - ie: service time for a kilobyte of data.  And you could
			// set an alarm on some combination of these metrics.
			//
			//By definition, spawning a new instance will spread the load.
			if cwStats.Load != nil {
				metricDatum = append(metricDatum,
					&cloudwatch.MetricDatum{
						MetricName: aws.String("srv/load"),
						Dimensions: dims,
						Timestamp:  now,
						Unit:       aws.String("None"),
						Value:      cwStats.Load,
					},
				)
			}

			//Log all outgoing data to the logstream for now
			for _, d := range metricDatum {
				logMetricDatum(logger, d)
			}

			//Log into a namespace containing our IP.  Because the purpose is to restart machines,
			//we log by machine, where odrive is a dimension on it
			params := &cloudwatch.PutMetricDataInput{
				Namespace:  namespace,
				MetricData: metricDatum,
			}

			if len(metricDatum) > 0 {
				if cwSession != nil {
					_, err := cwSession.PutMetricData(params)
					if err != nil {
						logger.Warn("cloudwatch put metric data fail", zap.Error(err))
					} else {
						logger.Debug("cloudwatch success")
					}
				}
			}
		}
	}()
}

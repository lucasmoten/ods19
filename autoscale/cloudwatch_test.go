package autoscale_test

import (
	"testing"

	"github.com/deciphernow/object-drive-server/autoscale"
	performance "github.com/deciphernow/object-drive-server/performance"
	"github.com/deciphernow/object-drive-server/util"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

//This gives us what looks like two probes of /proc/stats a second apart,
//and testing this way allows us to not actually use Linux or real-time for the tests
//
//Note: this bypasses the *parsing* of actual values from /proc!
func generateStats() (prevStat *linuxproc.Stat, nextStat *linuxproc.Stat, loadAvg *autoscale.LoadAvgStat) {
	return &linuxproc.Stat{
			CPUStats: []linuxproc.CPUStat{
				linuxproc.CPUStat{
					User:      121135,
					Nice:      0,
					System:    355884,
					Idle:      1317729,
					IOWait:    16963,
					IRQ:       0,
					SoftIRQ:   16675,
					Steal:     0,
					Guest:     0,
					GuestNice: 0,
				},
			},
		}, &linuxproc.Stat{
			CPUStats: []linuxproc.CPUStat{
				linuxproc.CPUStat{
					User:      121136,
					Nice:      0,
					System:    355907,
					Idle:      1318307,
					IOWait:    16963,
					IRQ:       0,
					SoftIRQ:   16675,
					Steal:     0,
					Guest:     0,
					GuestNice: 0,
				},
			},
		}, &autoscale.LoadAvgStat{
			CPU1Min:          0.51,
			CPU5Min:          0.40,
			CPU10Min:         0.36,
			RunningProcesses: 5,
			TotalProcesses:   804,
			LastPid:          47,
		}
}

func TestProcStat(t *testing.T) {
	//Note: we need to mock out /proc related structs and data because those ONLY exist on Linux!

	//Simulate a pair of measurements over a virtual interval
	tracker := performance.NewJobReporters(64)
	now := util.NowMS()
	autoscale.CloudWatchStartInterval(tracker, now)
	prevStat, nextStat, loadStat := generateStats()
	//Simulate having latency and throughput data to report: 1500 bytes in 500ms
	autoscale.CloudWatchTransactionRaw(now, now+500, 1500)
	s, _ := autoscale.ComputeOverallPerformance(prevStat, nextStat, loadStat, now+500)

	//Note: latency and throughput could still be nil due to tracker info
	//being absorbed via a channel in a goroutine
	t.Logf(
		"latency=%f throughput=%v cpuUtilization=%f pct memKB=%f kB memPct=%f pct load=%f",
		*s.Latency,
		*s.Throughput,
		*s.CPUUtilization,
		*s.MemKB,
		*s.MemPct,
		*s.Load,
	)

	if *s.CPUUtilization < 0.0 {
		t.Fatalf("utilization cannot be below zero: %v", *s.CPUUtilization)
	}
	if *s.MemPct > 100.0 {
		t.Fatalf("memory cannot go above 100 pct: %v", *s.MemPct)
	}
	if *s.MemPct < 0.0 {
		t.Fatalf("memory cannot be below zero: %v", *s.MemPct)
	}
	if *s.Throughput < 0.0 {
		t.Fatalf("throughput cannot be below zero: %v", *s.Throughput)
	}
	if *s.Load < 0.0 {
		t.Fatalf("load cannot be below zero: %v", *s.Load)
	}
}

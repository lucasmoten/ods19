package libs

import (
	"log"
	"time"
)

// ReporterID is used to locate our various counters
type ReporterID int

const (
	// UploadCounter handles counts for uploading
	UploadCounter = ReporterID(7)
	// DownloadCounter handles counts for downloading
	DownloadCounter = ReporterID(13)
)

/*
  Before an upload or download (or anything that is counted) begins,

  * we must get a signal that it is starting
    - pass in a filename ( for reference counting purposes )
    - ask this object to return us a start time.
    - implies that we can't start count until we know filename,
      which is partially through processing the request
  * when the counted thing finishes, we must give a report of
    - start,stop,bytes, filename

  It is important that timestamps be generated in the goroutine
  so that races do not cause the reports to come in with orders
  that break the ability to efficiently process events.

  Currently, the start is Unix time (in seconds)

  Given this, we are then able to compute the throughput of individual
  transactions:

  * bytes/(stop-start) ==> throughput in units of B/s

  Given that we report when a task starts, and giving a name,
  it is now possible to always know the load
  (ie: population count, or queue length), and to list jobs currently
  in progress.

  Reporting statistics requires of the list of JobReport:
  * if Start > Stop, then this is the mark of the beginning of a job
  * if Start <= Stop, then it marks the end of a job
  * Start of job beginning must be less than or equal to Start of
    job beginnings that come later
  * Stop of a job must match up with the Start of an end job
*/

// BeganJob is an opaque timestamp and other things eventually
type BeganJob int64

// EndedJob is an opaque timestamp used for internal calculations
type EndedJob int64

// SizeJob is the number of bytes to report
type SizeJob int64

// JobReport is the atomic unit of reporting progress into reporters
//Time units are opaque to the user of this API.
type JobReport struct {
	Start           BeganJob
	Stop            EndedJob
	SizeJob         SizeJob
	PopulationStart int
	PopulationStop  int
	JobName         string
}

// EndingJob is used to signal that a job is ending
type EndingJob struct {
	JobReport  JobReport
	ReporterID ReporterID
}

// CanDeleteHandler is invoked to remove cached files, we might
// want to delete it immediately.  There is currently
// a race condition until we can be certain that there
// are no readers/writers on it in between getting this signal
// and doing the delete
type CanDeleteHandler func(jobName string)

// PanicOnProblem is true during unit testing
var PanicOnProblem bool

// JobReporters is the collection of JobReporter objects that
//share ref counts on files.
//
// When a job name ref count goes to zero, put it in the list for
// deleting.  We do not want to download and delete at the same time.
type JobReporters struct {
	CreateTime       EndedJob
	Reporters        map[ReporterID]*JobReporter
	JobNameRefCount  map[string]int
	JobNameDeleting  []string
	BeginningJob     chan BeginningJob
	BeganJob         chan BeganJob
	EndingJob        chan EndingJob
	RequestingReport chan RequestingReport
	RequestedReport  chan RequestedReport
	Quit             chan int
	CanDelete        chan string
	CanDeleteHandler CanDeleteHandler
}

// BeginningJob is a request to the goroutine to generate a
// timestamp for the start of the job
type BeginningJob struct {
	JobName    string
	ReporterID ReporterID
}

// RequestingReport is the message that asks for a report
type RequestingReport struct {
	ReporterID ReporterID
}

// RequestedReport is enough information to display the report
type RequestedReport struct {
	Name                         string
	Size                         int64
	Duration                     int64
	PopulationWeightedByDuration int64
}

// ReportsQueue is a specialized queue for reports
// - Tail is where we can read the last pushed element
type ReportsQueue struct {
	Reports  []JobReport
	Capacity int
	Head     int
	Tail     int
}

// NewReportsQueue creates a queue for new reports
// It is a circular queue that quietly drops when it is at capacity,
// and items pushed in are reconciled in a way that is specific to
// JobReports.
//
// When reconciled, at every time period, the population and throughput
// are well defined.
func NewReportsQueue(capacity int) *ReportsQueue {
	return &ReportsQueue{
		Reports:  make([]JobReport, capacity),
		Head:     0,
		Tail:     capacity - 1,
		Capacity: capacity,
	}
}

var verbose = false

// PushTail moves cursor to write into the tail
// Write into the tail after this.
//
// It then reconciles the records so that throughput
// for intervals accurately accounts for concurrency,
// by effectively subtracting out double-counted time
// and splitting records to reflect overlaps.
func (r *ReportsQueue) PushTail(jr JobReport) {
	if (r.Tail+2)%r.Capacity == r.Head {
		r.Head++
		r.Head %= r.Capacity
		if int64(jr.Start) > int64(jr.Stop) {
			log.Printf("")
			log.Printf("WARNING: we lost track of when an upload began")
			if PanicOnProblem {
				panic("make the window larger")
			}
		}
	}
	r.Tail++
	r.Tail %= r.Capacity
	r.Reports[r.Tail] = jr
	if verbose {
		log.Printf("appending %v", r)
	}
	if r.Tail == r.Head {
		//There is only one entry
	} else {
		i := r.Tail
		for {
			ai := int64(r.Reports[i].Start)
			bi := int64(r.Reports[i].Stop)
			ci := int64(r.Reports[i].SizeJob)
			//Previous item-- circularly
			j := ((i - 1) + r.Capacity) % r.Capacity
			aj := int64(r.Reports[j].Start)
			bj := int64(r.Reports[j].Stop)
			cj := int64(r.Reports[j].SizeJob)
			if ai < bi && aj > bj {
				//if ai==bi we get divide by zero
				//bi cannot be edited, because that's tied to population change
				//
				// .. .. ..     - even earlier reports
				// aj bj cj     (start, stop, count) - earlier report
				// ai bi ci     (start, stop, count) - later report
				//
				// Keep the portion of events from bi to aj in ci, push remainder to j
				//
				// example: (with population count included as di,dj)
				//   0 0 0   0
				//   4 0 0   1
				//   5 0 0   2     row j
				//   4 9 10  1     row i
				//
				//  Something began at time 5, and we don't know when it ended.
				//  But since 4 9 10 came in at time 9, we know that the job at
				//  time 5 must end at time 9 or later.
				//
				//  A job from 4 to 9 had 10 events.  So we split up the data
				//  to reflect actual throughput for the timespan:
				//   d = ci*(bi-aj)/(bi-aj)
				//   d = 10*(9-5)/(9-4) = 10*4/5 = 40/5 = 8
				//  the integer divide is intentional so that we don't lose data,
				//  at the cost of slightly jittering the original throughput:
				//  ad <= ci
				//
				//  as jobs end, they adopt the population of the entry before them
				//
				//  4 0 0  1
				//  5 4 2  2  //needs a swap before stopping - bj should not be zero unless ai was
				//  5 9 8  1
				//
				//  4 0 0  1
				//  4 5 2  2  * continue this algorithm from this point
				//  5 9 8  1
				//
				// Note case:
				//      4 0 0
				//      5 0 0
				//      5 9 10
				d := ci * (bi - aj) / (bi - ai)
				r.Reports[j].Stop = EndedJob(ai)
				r.Reports[i].Start = BeganJob(aj)
				r.Reports[j].SizeJob += SizeJob(ci - d)
				r.Reports[i].SizeJob = SizeJob(d)
				//				r.Reports[i].Population = r.Reports[j].Population
				if r.Reports[j].SizeJob < 0 || r.Reports[i].SizeJob < 0 {
					log.Printf("%v", r)
					if PanicOnProblem {
						panic("impossible state")
					}
				}
				//Swap start and stop for row j if they are now inverted
				if int64(r.Reports[j].Start) > int64(r.Reports[j].Stop) {
					r.Reports[j].Stop = EndedJob(r.Reports[j].Start)
					r.Reports[j].Start = BeganJob(ai)
				}
				//row i no longer associated with the entire download
				//and row j is related to this one in addition to whatever else it was
				r.Reports[i].JobName = ""
				if verbose {
					log.Printf("after rebuild1 %v", r)
				}
			} else {
				//We have two records overlapping in end record, but start are in order
				if ai < bi && aj < bj && ai <= bj {
					//if ai==bi we get divide by zero
					//bi cannot be edited, because that's tied to population change
					//
					//4  9 24
					//6 12 13
					//
					// d = 13*(12-9)/(12-6) = 13*3/6 = 39/6 = 6
					//
					//4  9 31
					//9 12 6
					d := ci * (bi - bj) / (bi - ai)
					r.Reports[j].SizeJob += SizeJob(ci - d)
					r.Reports[i].Start = BeganJob(bj)
					//r.Reports[i].Population = r.Reports[j].Population
					r.Reports[i].JobName = ""
					if verbose {
						log.Printf("after rebuild2 %v", r)
					}
				} else {
					if ai > bi {
						//ignore it...this is just appending start markers
					} else {
						if aj == bj && cj == 0 {
							//ignore it as it's just where a population count changes
						} else {
							if ai == bi && ci == 0 {
								/////Leaving zero intervals null
								r.Reports[i].Start = BeganJob(r.Reports[j].Stop)
								break
							} else {
								log.Printf("unhandled: i:%d %d %d, j:%d %d %d", ai, bi, ci, aj, bj, cj)
								if PanicOnProblem {
									panic("unhandled case")
								}
							}
						}
					}
				}
			}
			//Previous item - circularly
			if i == r.Head {
				break
			}
			i = ((i - 1) + r.Capacity) % r.Capacity
		}
	}
}

// PopHead discards the first element of the queue
// Read that element out of head before you pop
func (r *ReportsQueue) PopHead() JobReport {
	retval := r.Reports[r.Head]
	r.Head++
	r.Head %= r.Capacity
	return retval
}

// PeekHead gets the item in head of queue
func (r *ReportsQueue) PeekHead() JobReport {
	return r.Reports[r.Head]
}

// PeekTail gets the item in the tail of the queue
func (r *ReportsQueue) PeekTail() JobReport {
	return r.Reports[r.Tail]
}

// Empty checks to see if there is data in the queue
func (r *ReportsQueue) Empty() bool {
	return ((r.Tail+1)%r.Capacity == r.Head)
}

// JobReporter is an individual counter
type JobReporter struct {
	Name              string
	Q                 *ReportsQueue
	ReporterID        ReporterID
	TotalBytes        int64
	TotalTime         int64
	PopWeightedByTime int64
}

func getTStamp() int64 {
	return (time.Now().UnixNano() / (1000 * 1000))
}

func jobReportersBeginning(r *JobReporters, beginningJob BeginningJob) BeganJob {
	beganJob := BeganJob(getTStamp())

	reporter := r.Reporters[beginningJob.ReporterID]

	jobReport := JobReport{
		Start: beganJob,
		//The lowest possible timestamp that is within our observation period
		Stop:    r.CreateTime,
		SizeJob: SizeJob(0),
	}

	var prevPopulation = 0
	if reporter.Q.Empty() == false {
		prevPopulation = reporter.Q.PeekTail().PopulationStop
	}
	jobReport.PopulationStart = prevPopulation + 1
	jobReport.PopulationStop = prevPopulation + 1

	reporter.Q.PushTail(jobReport)
	r.JobNameRefCount[beginningJob.JobName]++

	return beganJob
}

func jobReportersJobReport(r *JobReporters, j EndingJob) {
	//Snap to millisecond
	j.JobReport.Stop = EndedJob(getTStamp())
	duration := int64(j.JobReport.Stop) - int64(j.JobReport.Start)

	//Increment the counters
	reporter := r.Reporters[j.ReporterID]
	reporter.TotalTime += duration
	reporter.TotalBytes += int64(j.JobReport.SizeJob)

	var prevPopulation = reporter.Q.PeekTail().PopulationStop
	if prevPopulation < 0 {
		log.Printf("r:%v", r)
		panic("we cannot call this with a zero or less population!")
	}
	j.JobReport.PopulationStart = prevPopulation - 1
	j.JobReport.PopulationStop = prevPopulation - 1
	reporter.Q.PushTail(j.JobReport)

	reporter.PopWeightedByTime += duration * int64(j.JobReport.PopulationStart)

	//Decrement the reference count on this file
	r.JobNameRefCount[j.JobReport.JobName]--
	if r.JobNameRefCount[j.JobReport.JobName] == 0 {
		r.CanDelete <- j.JobReport.JobName
	}
}

func jobReportersRequestingReport(reporters *JobReporters, requestingReport RequestingReport) RequestedReport {
	r := reporters.Reporters[requestingReport.ReporterID]
	rr := RequestedReport{
		Size:     r.TotalBytes,
		Duration: r.TotalTime,
		Name:     r.Name,
		PopulationWeightedByDuration: r.PopWeightedByTime,
	}
	return rr
}

// This is the goroutine that must absorb all reporting on
// when counted transactions start and stop.
// All in/out channels are used here
func jobReportersThread(r *JobReporters) {
	for {
		select {
		case beginningJob := <-r.BeginningJob:
			r.BeganJob <- jobReportersBeginning(r, beginningJob)
		case endingJob := <-r.EndingJob:
			jobReportersJobReport(r, endingJob)
		case requestingReport := <-r.RequestingReport:
			r.RequestedReport <- jobReportersRequestingReport(r, requestingReport)
		case _ = <-r.Quit:
			return
		}
	}
}

// NewJobReporters is where the counters live.  They share
// the ref counts on files in progress.
//
//  Channels are buffered to allow for async progress
func NewJobReporters(canDeleteHandler CanDeleteHandler) *JobReporters {
	reporters := &JobReporters{
		CreateTime: EndedJob(getTStamp()),
		Reporters:  make(map[ReporterID]*JobReporter),
		//this is why the channels are shared
		JobNameRefCount:  make(map[string]int),
		BeginningJob:     make(chan BeginningJob, 10),
		BeganJob:         make(chan BeganJob, 10),
		EndingJob:        make(chan EndingJob, 10),
		RequestingReport: make(chan RequestingReport, 10),
		RequestedReport:  make(chan RequestedReport, 10),
		Quit:             make(chan int),
		CanDelete:        make(chan string, 20),
		CanDeleteHandler: canDeleteHandler,
	}
	reporters.Reporters[UploadCounter] = reporters.makeReporter("upload")
	reporters.Reporters[DownloadCounter] = reporters.makeReporter("download")

	//Listen in on job reports
	go jobReportersThread(reporters)

	//Delete files that are elegible for deletion
	go func() {
		for {
			toDelete := <-reporters.CanDelete
			reporters.CanDeleteHandler(toDelete)
		}
	}()

	return reporters
}

// Stop the goroutine
func (jrs *JobReporters) Stop() {
	jrs.Quit <- 0
}

func (jrs *JobReporters) makeReporter(name string) *JobReporter {
	capacity := 128
	reporter := &JobReporter{
		Name: name,
		Q:    NewReportsQueue(capacity),
	}
	return reporter
}

// BeginTime gets us a timestamp to use for reporting end time
func (jrs *JobReporters) BeginTime(reporterID ReporterID, jobName string) BeganJob {
	jrs.BeginningJob <- BeginningJob{
		ReporterID: reporterID,
		JobName:    jobName,
	}
	return <-jrs.BeganJob
}

// EndTime reports how much data was transferred.
// Note that we can delete this file
// BUG(000) there is a rare race condition until we figure out how to delete files
// without blocking the goroutine.  We want to block reading or writing the file
// until the delete finishes, based on the data structures in the goroutine.
func (jrs *JobReporters) EndTime(reporterID ReporterID, start BeganJob, jobName string, sizeJob SizeJob) {
	jrs.EndingJob <- EndingJob{
		ReporterID: reporterID,
		JobReport: JobReport{
			Start:   start,
			JobName: jobName,
			SizeJob: sizeJob,
		},
	}
}

// Report gives us a report of current statistics
func (jrs *JobReporters) Report(reporterID ReporterID) RequestedReport {
	jrs.RequestingReport <- RequestingReport{
		ReporterID: reporterID,
	}
	return <-jrs.RequestedReport
}

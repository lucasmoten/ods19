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
	Start      BeganJob
	Stop       EndedJob
	SizeJob    SizeJob
	Population int
	JobName    string
}

// EndingJob is used to signal that a job is ending
type EndingJob struct {
	JobReport  JobReport
	ReporterID ReporterID
}

// JobReporters is the collection of JobReporter objects that
//share ref counts on files.
//
// When a job name ref count goes to zero, put it in the list for
// deleting.  We do not want to download and delete at the same time.
type JobReporters struct {
	Reporters        map[ReporterID]*JobReporter
	JobNameRefCount  map[string]int
	JobNameDeleting  []string
	BeginningJob     chan BeginningJob
	BeganJob         chan BeganJob
	EndingJob        chan EndingJob
	RequestingReport chan RequestingReport
	RequestedReport  chan RequestedReport
	Quit             chan int
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
func NewReportsQueue(capacity int) *ReportsQueue {
	return &ReportsQueue{
		Reports:  make([]JobReport, capacity),
		Head:     0,
		Tail:     capacity - 1,
		Capacity: capacity,
	}
}

// PushTail moves cursor to write into the tail
// Write into the tail after this.
func (r *ReportsQueue) PushTail(jobReport JobReport) {
	r.Tail++
	r.Tail %= r.Capacity
	r.Reports[r.Tail] = jobReport
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

func jobReportersBeginning(r *JobReporters, beginningJob BeginningJob) BeganJob {
	beganJob := BeganJob(time.Now().UnixNano() / (1000 * 1000))

	reporter := r.Reporters[beginningJob.ReporterID]

	jobReport := JobReport{
		Start:   beganJob,
		Stop:    EndedJob(0),
		SizeJob: SizeJob(0),
	}

	var prevPopulation = 0
	if reporter.Q.Empty() == false {
		prevPopulation = reporter.Q.PeekTail().Population
	}
	jobReport.Population = prevPopulation + 1

	reporter.Q.PushTail(jobReport)
	r.JobNameRefCount[beginningJob.JobName]++

	return beganJob
}

func jobReportersJobReport(r *JobReporters, j EndingJob) {
	//Snap to millisecond
	j.JobReport.Stop = EndedJob(time.Now().UnixNano() / (1000 * 1000))
	duration := int64(j.JobReport.Stop) - int64(j.JobReport.Start)

	//Increment the counters
	reporter := r.Reporters[j.ReporterID]
	reporter.TotalTime += duration
	reporter.TotalBytes += int64(j.JobReport.SizeJob)

	var prevPopulation = reporter.Q.PeekTail().Population
	if prevPopulation < 1 {
		log.Printf("r:%v", r)
		panic("we cannot call this with a zero or less population!")
	}
	j.JobReport.Population = prevPopulation - 1
	reporter.Q.PushTail(j.JobReport)

	reporter.PopWeightedByTime += duration * int64(j.JobReport.Population)

	//Decrement the reference count on this file
	r.JobNameRefCount[j.JobReport.JobName]--
	if r.JobNameRefCount[j.JobReport.JobName] == 0 {
		log.Printf("canDelete:%s", j.JobReport.JobName)
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
func NewJobReporters() *JobReporters {
	reporters := &JobReporters{
		Reporters:        make(map[ReporterID]*JobReporter),
		JobNameRefCount:  make(map[string]int),
		BeginningJob:     make(chan BeginningJob, 10),
		BeganJob:         make(chan BeganJob, 10),
		EndingJob:        make(chan EndingJob, 10),
		RequestingReport: make(chan RequestingReport, 10),
		RequestedReport:  make(chan RequestedReport, 10),
		Quit:             make(chan int),
	}
	reporters.Reporters[UploadCounter] = reporters.makeReporter("upload")
	reporters.Reporters[DownloadCounter] = reporters.makeReporter("download")
	go jobReportersThread(reporters)
	return reporters
}

// Stop the goroutine
func (jrs *JobReporters) Stop() {
	jrs.Quit <- 0
}

func (jrs *JobReporters) makeReporter(name string) *JobReporter {
	capacity := 512
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

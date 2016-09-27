package performance

import (
	"fmt"
	"io"
	"log"
	"time"
)

// ReporterID is used to locate our various counters
type ReporterID int

const (
	// UploadCounter handles counts for uploading
	UploadCounter = ReporterID(1)
	// DownloadCounter handles counts for downloading
	DownloadCounter = ReporterID(2)
	// S3DrainTo is the time to drain a file back into S3
	S3DrainTo = ReporterID(3)
	// S3DrainFrom is time to recache file
	S3DrainFrom = ReporterID(4)
	// AACCounterCheckAccess tracking for if user has access to an ACM
	AACCounterCheckAccess = ReporterID(5)
	// AACCounterGetSnippets tracking for fetching user snippets and formatting
	AACCounterGetSnippets = ReporterID(6)
	// AACCounterPopulateAndValidateResponse tracking each call to populate flattened acm response
	AACCounterPopulateAndValidateResponse = ReporterID(7)
	// AACCounterCheckAccessAndPopulate tracking each call to check access and populate the flattened fields of acm
	AACCounterCheckAccessAndPopulate = ReporterID(8)
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
	Start   BeganJob
	Stop    EndedJob
	SizeJob SizeJob
}

// EndingJob is used to signal that a job is ending
type EndingJob struct {
	JobReport  JobReport
	ReporterID ReporterID
}

// PanicOnProblem is true during unit testing
var PanicOnProblem bool

// JobReporters is the collection of JobReporter objects that
//share ref counts on files.
//
// When a job name ref count goes to zero, put it in the list for
// deleting.  We do not want to download and delete at the same time.
type JobReporters struct {
	Capacity         int
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

	RecentRequestCount int64
	RecentByteCount    int64
	RecentBegin        int64
}

func (j *JobReporters) NewInterval() {
	j.RecentRequestCount = 0
	j.RecentByteCount = 0
	j.RecentBegin = getTStampMS()
}

// BeginningJob is a request to the goroutine to generate a
// timestamp for the start of the job
type BeginningJob struct {
	ReporterID ReporterID
}

// RequestingReport is the message that asks for a report
type RequestingReport struct {
	ReporterID ReporterID
}

// RequestedReport is enough information to display the report
// TODO: make this be an array of such structs to handle load
// dependent throughput calculations
//
//  with a bucket for each population range: (1 << n) ... ((1 << (n+1))-1)
//
type RequestedReport struct {
	Name                         string
	Size                         int64
	Duration                     int64
	PopulationWeightedByDuration int64
}

// QEntry - entries in the queue where logged events are reconciled
type QEntry struct {
	TStamp     int64
	Population int64
	Bytes      int64
}

// QStat Statistics for items that aged out of the queue
type QStat struct {
	TotalTime         int64
	TotalBytes        int64
	PopWeightedByTime int64
}

// ReportsQueue is a specialized queue for reports
// - Tail is where we can read the last pushed element
type ReportsQueue struct {
	RequestedReport RequestedReport
	Entry           []QEntry
	Capacity        int
	Head            int
	Tail            int
	Stat            QStat
	UserStat        QStat
	NumeratorUnits  string
	Name            string
}

// NewReportsQueue creates a queue for new reports
// It is a circular queue that quietly drops when it is at capacity,
// and items pushed in are reconciled in a way that is specific to
// JobReports.
//
// When reconciled, at every time period, the population and throughput
// are well defined.
func NewReportsQueue(name string, capacity int, numeratorUnits string) *ReportsQueue {
	q := &ReportsQueue{
		Name:           name,
		Entry:          make([]QEntry, capacity),
		Head:           0,
		Tail:           capacity - 1,
		Capacity:       capacity,
		NumeratorUnits: numeratorUnits,
	}
	return q
}

var verbose = false

//AdvanceHead the head even though we are full
//Absorb lost information into stats counters
func (r *ReportsQueue) AdvanceHead() {
	//If the queue is full
	if ((r.Tail + 2) % r.Capacity) == r.Head {
		//Dump off the head into statistics
		i := r.Head
		j := (r.Head + 1) % r.Capacity
		t := r.Entry[j].TStamp - r.Entry[i].TStamp
		r.Stat.TotalTime += t
		r.Stat.PopWeightedByTime += t * r.Entry[i].Population
		r.Stat.TotalBytes += r.Entry[i].Bytes
		r.Head++
		r.Head %= r.Capacity
	}
}

// InsertStat stuffs reports into the statistics queue
// The reason we have the queue is that we need to reconcile time overlaps
// to remove double-counted time.  Otherwise, we will only be
// getting statistics that reflect the individual transactions as experienced
// by the user.  We also need numbers that reflect aggregate throughput
// due to concurrency.
func (r *ReportsQueue) InsertStat(jr JobReport) {

	var eOld = &r.Entry[r.Tail]
	var amountToDistribute = int64(0)

	beginAt := int64(jr.Start)
	endAt := int64(jr.Stop)
	size := int64(jr.SizeJob)

	interval := endAt - beginAt

	if interval == 0 {
		log.Printf("%v", jr)
		panic("we cannot work with intervals of length 0")
	}

	// negative interval indicates starting a txn only.
	if beginAt > 0 && endAt == 0 {
		if eOld.TStamp < beginAt {
			// don't lose stats when we advance tail over head
			r.AdvanceHead()
			r.Tail++
			r.Tail %= r.Capacity
			eCurrent := &r.Entry[r.Tail]
			eCurrent.TStamp = beginAt
			eCurrent.Population = eOld.Population + 1
		} else {
			//stacking multiple starts
			if eOld.TStamp == beginAt {
				eOld.Population++
			} else {
				//1ms job case
				ePrev := &r.Entry[(r.Tail+r.Capacity-1)%r.Capacity]
				if ePrev.TStamp == beginAt {
					ePrev.Population++
					eOld.Population++
				} else {
					//End then begin at same timestamp
					if eOld.TStamp-1 == beginAt {
						eOld.Population++
					} else {
						//beginAt < eOld.TStamp && endAt == 0
						log.Printf("ERROR: %d < %d && %d == 0", beginAt, eOld.TStamp, endAt)
						if PanicOnProblem {
							panic("bad state")
						}
					}
				}
			}
		}
	} else {
		// a txn has completed in these cases
		if eOld.TStamp == endAt {
			eOld.Population--
			if eOld.Population < 0 {
				log.Printf("%v", r)
				if PanicOnProblem {
					panic("pop went below zero")
				}
			}
			amountToDistribute = size
			r.UserStat.TotalTime += interval
			r.UserStat.TotalBytes += size
			r.UserStat.PopWeightedByTime += interval * eOld.Population
		} else {
			// don't lose stats when we advance tail over head
			r.AdvanceHead()
			r.Tail++
			r.Tail %= r.Capacity
			eNext := &r.Entry[r.Tail]
			eNext.TStamp = endAt
			eNext.Population = eOld.Population - 1
			if eNext.Population < 0 {
				if PanicOnProblem {
					panic("pop went below zero")
				}
			}
			amountToDistribute = size
			r.UserStat.TotalTime += interval
			r.UserStat.TotalBytes += size
			r.UserStat.PopWeightedByTime += interval * eNext.Population
		}
	}

	//Proportionally distribute bytes across the time period
	i := r.Tail
	for {
		if amountToDistribute < 0 {
			if PanicOnProblem {
				log.Printf("cannot redistribute %dB", amountToDistribute)
				panic("cannot redistribute")
			}
		}
		if amountToDistribute == 0 {
			break
		}
		j := (i + r.Capacity - 1) % r.Capacity
		if r.Entry[j].TStamp <= beginAt {
			r.Entry[j].Bytes += amountToDistribute
			amountToDistribute = 0
			break
		}
		ourInterval := r.Entry[i].TStamp - r.Entry[j].TStamp
		d := amountToDistribute * ourInterval / interval
		r.Entry[j].Bytes += d
		amountToDistribute -= d
		i = (i + r.Capacity - 1) % r.Capacity
	}
}

// Length of the queue
func (r *ReportsQueue) Length() int {
	if r.Head == (r.Tail+1)%r.Capacity {
		return 0
	}
	if r.Head < r.Tail {
		return r.Tail - r.Head + 1
	}
	return (r.Tail - r.Head + 1 + r.Capacity)
}

// Dump shows the internal state of the queue
func (r *ReportsQueue) Dump(w io.Writer, verbose bool) {
	if r.Length() < 1 {
		return
	}

	var maxPop = int64(0)

	i := r.Tail
	units := r.NumeratorUnits
	for {
		j := (i + r.Capacity - 1) % r.Capacity
		//This is supposedly impossible to be zero....
		t := (r.Entry[i].TStamp - r.Entry[j].TStamp)
		b := r.Entry[j].Bytes
		if t > 0 && b > 0 && verbose {
			fmt.Fprintf(
				w,
				"%d Q %v %s/s => %v %s in %v ms\n",
				r.Entry[j].Population,
				(1000.0*r.Entry[j].Bytes)/t,
				units,
				r.Entry[j].Bytes,
				units,
				r.Entry[i].TStamp-r.Entry[j].TStamp,
			)
		}
		if r.Entry[j].Population > int64(maxPop) {
			maxPop = r.Entry[j].Population
		}
		if j == r.Head {
			break
		}
		i = (i + r.Capacity - 1) % r.Capacity
	}

	entryPerPop := make([]QStat, maxPop+1)

	i = r.Tail
	//Calculate throughput per population
	for {
		j := (i + r.Capacity - 1) % r.Capacity
		t := r.Entry[i].TStamp - r.Entry[j].TStamp
		b := r.Entry[j].Bytes
		w := r.Entry[j].Population * t
		p := r.Entry[j].Population
		entryPerPop[p].TotalBytes += b
		entryPerPop[p].TotalTime += t
		entryPerPop[p].PopWeightedByTime += w

		if j == r.Head {
			break
		}
		i = (i + r.Capacity - 1) % r.Capacity
	}

	fmt.Fprintf(w, "\n%s Throughput By Queue Length:\n", r.Name)
	//Show throughput per population
	for p := int64(1); p <= maxPop; p++ {
		if entryPerPop[p].TotalTime > 0 {
			fmt.Fprintf(
				w,
				"%d Q %v %s/s for %d s\n",
				p,
				(1000*entryPerPop[p].TotalBytes)/entryPerPop[p].TotalTime,
				units,
				entryPerPop[p].TotalTime/1000,
			)
		}
	}

	if r.Stat.TotalTime > 0 {
		fmt.Fprintf(
			w,
			"%s flushed: %v Q %v %s/s => %v %s in %v ms\n",
			r.Name,
			float32(r.Stat.PopWeightedByTime)/float32(r.Stat.TotalTime),
			(1000*r.Stat.TotalBytes)/r.Stat.TotalTime,
			units,
			r.Stat.TotalBytes,
			units,
			r.Stat.TotalTime,
		)
	}
	if r.UserStat.TotalTime > 0 {
		fmt.Fprintf(
			w,
			"%s userExperience: %v Q %v %s/s => %v %s in %v ms\n",
			r.Name,
			float32(r.UserStat.PopWeightedByTime)/float32(r.UserStat.TotalTime),
			(1000*r.UserStat.TotalBytes)/r.UserStat.TotalTime,
			units,
			r.UserStat.TotalBytes,
			units,
			r.UserStat.TotalTime,
		)
	}

}

// PeekTail gets the item in the tail of the queue
func (r *ReportsQueue) PeekTail() QEntry {
	return r.Entry[r.Tail]
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

func getTStampMS() int64 {
	return (time.Now().UnixNano() / (1000 * 1000))
}

func jobReportersBeginning(r *JobReporters, beginningJob BeginningJob) BeganJob {
	beganJob := BeganJob(getTStampMS())

	reporter := r.Reporters[beginningJob.ReporterID]

	jobReport := JobReport{
		Start: beganJob,
		//The lowest possible timestamp that is within our observation period
		Stop:    r.CreateTime,
		SizeJob: SizeJob(0),
	}

	reporter.Q.InsertStat(jobReport)

	return beganJob
}

func jobReportersJobReport(r *JobReporters, j EndingJob) {
	//Snap to millisecond - notice that end stamps always are ahead by 1, to make
	//divide by zero impossible.
	j.JobReport.Stop = EndedJob(getTStampMS() + int64(1))
	duration := int64(j.JobReport.Stop) - int64(j.JobReport.Start)

	//Increment the counters
	reporter := r.Reporters[j.ReporterID]
	reporter.TotalTime += duration
	reporter.TotalBytes += int64(j.JobReport.SizeJob)

	reporter.Q.InsertStat(j.JobReport)
}

func jobReportersRequestingReport(reporters *JobReporters, requestingReport RequestingReport) RequestedReport {
	r := reporters.Reporters[requestingReport.ReporterID]
	rr := RequestedReport{
		Size:     r.Q.Stat.TotalBytes,
		Duration: r.Q.Stat.TotalTime,
		Name:     r.Name,
		PopulationWeightedByDuration: r.Q.Stat.PopWeightedByTime,
	}
	return rr
}

// This is the goroutine that must absorb all reporting on
// when counted transactions start and stop.
// All in/out channels are used here
func jobReportersThread(r *JobReporters) {
	for {
		select {
		case endingJob := <-r.EndingJob:
			jobReportersJobReport(r, endingJob)
		case beginningJob := <-r.BeginningJob:
			r.BeganJob <- jobReportersBeginning(r, beginningJob)
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
func NewJobReporters(capacity int) *JobReporters {
	reporters := &JobReporters{
		Capacity:   capacity,
		CreateTime: EndedJob(0),
		Reporters:  make(map[ReporterID]*JobReporter),
		//this is why the channels are shared
		JobNameRefCount:  make(map[string]int),
		BeginningJob:     make(chan BeginningJob, 32),
		BeganJob:         make(chan BeganJob, 32),
		EndingJob:        make(chan EndingJob, 32),
		RequestingReport: make(chan RequestingReport, 32),
		RequestedReport:  make(chan RequestedReport, 32),
		Quit:             make(chan int, 32),
	}
	reporters.Reporters[UploadCounter] = reporters.makeReporter("upload", "B")
	reporters.Reporters[DownloadCounter] = reporters.makeReporter("download", "B")
	reporters.Reporters[S3DrainTo] = reporters.makeReporter("s3drainto", "B")
	reporters.Reporters[S3DrainFrom] = reporters.makeReporter("s3drainfrom", "B")
	reporters.Reporters[AACCounterCheckAccess] = reporters.makeReporter("aaccheckaccess", "Req")
	reporters.Reporters[AACCounterGetSnippets] = reporters.makeReporter("aacgetsnippets", "Req")
	reporters.Reporters[AACCounterPopulateAndValidateResponse] = reporters.makeReporter("aacpopulateandvalidateresponse", "Req")
	reporters.Reporters[AACCounterCheckAccessAndPopulate] = reporters.makeReporter("aaccheckaccessandpopulate", "Req")

	//Listen in on job reports
	go jobReportersThread(reporters)

	return reporters
}

// Stop the goroutine
func (jrs *JobReporters) Stop() {
	jrs.Quit <- 0
}

func (jrs *JobReporters) makeReporter(name string, numeratorUnits string) *JobReporter {
	capacity := jrs.Capacity
	reporter := &JobReporter{
		Name: name,
		Q:    NewReportsQueue(name, capacity, numeratorUnits),
	}
	return reporter
}

// BeginTime gets us a timestamp to use for reporting end time
func (jrs *JobReporters) BeginTime(reporterID ReporterID) BeganJob {
	jrs.BeginningJob <- BeginningJob{
		ReporterID: reporterID,
	}
	return <-jrs.BeganJob
}

// EndTime reports how much data was transferred.
// Note that we can delete this file
// BUG(000) there is a rare race condition until we figure out how to delete files
// without blocking the goroutine.  We want to block reading or writing the file
// until the delete finishes, based on the data structures in the goroutine.
func (jrs *JobReporters) EndTime(reporterID ReporterID, start BeganJob, sizeJob SizeJob) {
	jrs.EndingJob <- EndingJob{
		ReporterID: reporterID,
		JobReport: JobReport{
			Start:   start,
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

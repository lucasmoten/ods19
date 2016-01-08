package libs

import (
	"log"
	"math/rand"
	"testing"
	"time"
)

var reporters *JobReporters

type job struct {
	ReporterID ReporterID
	Start      BeganJob
	FileName   string
}

func TestReportingThread_NonDeterministic(t *testing.T) {
	//A set of files to download and upload
	jobNames := []string{
		"chewbacca.jpg",
		"odrive.pdf",
		"ConcurrencyIsNotParallelism.mp4",
	}
	jobTypes := []ReporterID{
		UploadCounter,
		DownloadCounter,
	}

	log.Printf("Starting test")
	//Perform random actions on the API
	var started = make([]job, 0)
	var running = true
	//drop to zero population this many times
	rounds := 5
	for running {
		//Random sleep
		time.Sleep(time.Duration(rand.Int()%1000) * time.Millisecond)
		//Either start or finish a job
		if len(started) == 0 || (rand.Int()%10) > 4 {
			//Randomly up or down and random job name
			nm := jobNames[rand.Int()%len(jobNames)]
			jt := jobTypes[rand.Int()%len(jobTypes)]
			////The API call
			startedAt := reporters.BeginTime(jt, nm)
			job := job{
				Start:      startedAt,
				FileName:   nm,
				ReporterID: jt,
			}
			started = append(started, job)
			log.Printf("started %d:%s", job.ReporterID, job.FileName)
		} else {
			//Pick a random job to complete
			nth := rand.Int() % len(started)
			job := started[nth]
			started = append(started[:nth], started[nth+1:]...)
			size := SizeJob(rand.Int() % 1000000)
			////The API call
			reporters.EndTime(job.ReporterID, job.Start, job.FileName, size)
			log.Printf("ended %d:%s", job.ReporterID, job.FileName)
		}
		if len(started) == 0 {
			rounds--
			if rounds <= 0 {
				running = false
			}
		}
	}

	log.Printf("Getting reports")
	for i := 0; i < len(jobTypes); i++ {
		jobType := jobTypes[i]
		counter := reporters.Report(jobType)
		if counter.Duration > 0 {
			log.Printf(
				"%s:%d B/s, %f SessionAverage",
				counter.Name,
				counter.Size/counter.Duration,
				float32(counter.PopulationWeightedByDuration)/float32(counter.Duration),
			)
		}
		//Check invariants
		q := reporters.Reporters[jobType].Q
		h := q.Head
		t := q.Tail
		p := q.Reports[t].PopulationStop
		if p != 0 {
			log.Printf("%v", reporters.Reporters[jobType].Q)
			panic("inconsistent population")
		}
		idx := h
		for {
			if int64(q.Reports[idx].Start) > int64(q.Reports[idx].Stop) {
				log.Printf("%v", reporters.Reporters[jobType].Q)
				panic("start times should be before stop times")
			}
			prevIdx := idx
			idx = (idx + 1) % q.Capacity
			if int64(q.Reports[idx].Start) < int64(q.Reports[prevIdx].Stop) {
				log.Printf("%v", reporters.Reporters[jobType].Q)
				panic("start times of current should not be less than previous")
			}
			if idx == t {
				break
			}
		}
		//Dump the end result
		log.Printf("%v", reporters.Reporters[jobType].Q)
	}
	reporters.Stop()
}

func logPurge(name string) {
	log.Printf("files can be removed for %s", name)
}

func init() {
	PanicOnProblem = true
	reporters = NewJobReporters(logPurge)
}

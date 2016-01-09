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
	File       fileDescription
}

type fileDescription struct {
	Name string
	Size int64
}

func TestReportingThread_NonDeterministic(t *testing.T) {
	//A set of files to download and upload
	files := []fileDescription{
		fileDescription{"chewbacca.jpg", 10234},
		fileDescription{"grumptycat.jpg", 8214},
		fileDescription{"odrive.pdf", 90234},
		fileDescription{"ConcurrencyIsNotParallelism.mp4", 13000000},
		fileDescription{"everything.doc", 28385},
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
	rounds := 10
	for running {
		//Random sleep
		time.Sleep(time.Duration(rand.Int()%1000) * time.Millisecond)
		//Either start or finish a job
		if len(started) == 0 || (rand.Int()%10) > 4 {
			//Randomly up or down and random job name
			n := rand.Int() % len(files)
			jt := jobTypes[rand.Int()%len(jobTypes)]
			////The API call
			startedAt := reporters.BeginTime(jt, files[n].Name)
			job := job{
				Start:      startedAt,
				File:       files[n],
				ReporterID: jt,
			}
			started = append(started, job)
			log.Printf("started %d:%s", job.ReporterID, job.File.Name)
		} else {
			//Pick a random job to complete
			nth := rand.Int() % len(started)
			job := started[nth]
			started = append(started[:nth], started[nth+1:]...)
			////The API call
			reporters.EndTime(job.ReporterID, job.Start, job.File.Name, SizeJob(job.File.Size))
			log.Printf("ended %d:%s", job.ReporterID, job.File.Name)
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
		reporters.Reporters[jobType].InvariantsCheck()
		//Dump the end result
		//log.Printf("%v", reporters.Reporters[jobType].Q)
	}
	reporters.Stop()
}

func logPurge(name string) {
	log.Printf("files can be removed for %s", name)
}

func init() {
	PanicOnProblem = true
	reporters = NewJobReporters(32, logPurge)
}

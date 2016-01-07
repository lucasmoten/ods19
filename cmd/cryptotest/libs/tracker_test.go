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
	for running {
		//Random sleep
		time.Sleep(time.Duration(rand.Int()%300) * time.Millisecond)
		if (rand.Int()%10) > 5 || len(started) == 0 {
			//Randomly up or down and random job name
			nm := jobNames[rand.Int()%len(jobNames)]
			jt := jobTypes[rand.Int()%len(jobTypes)]
			job := job{
				Start:      reporters.BeginTime(jt, nm),
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
			reporters.EndTime(job.ReporterID, job.Start, job.FileName, size)
			log.Printf("ended %d:%s", job.ReporterID, job.FileName)
		}
		if rand.Int()%4 == 0 && len(started) == 0 {
			running = false
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
	}
	reporters.Stop()
}

func init() {
	reporters = NewJobReporters()
}

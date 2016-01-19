package performance

import (
	"log"
	"math/rand"
	"os"
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

//A set of files to download and upload
var files []fileDescription
var jobTypes []ReporterID

func simulate(i int, done chan int) {
	log.Printf("sim start:%d", i)

	//Pick a random job type and file
	n := rand.Int() % len(files)
	jt := jobTypes[rand.Int()%1]

	//Noise between 1.0 and 1.5
	var noise = 1.0 + float32(rand.Int()%500)/1000.0

	time.Sleep(time.Duration(rand.Int()%10000) * time.Millisecond)

	//and run for some jittery time proportional to file size
	//100kB/s with some noise
	var bandwidth float32 = 100000.0
	startedAt := reporters.BeginTime(jt, files[n].Name)
	//log.Printf(" began[%d]", i)

	transactionTime := noise * float32(files[n].Size) / bandwidth
	log.Printf("%d transactionTime will be %vs", i, transactionTime)
	//Do the actual sleep
	time.Sleep(time.Duration(transactionTime) * time.Second)
	reporters.EndTime(jt, startedAt, files[n].Name, SizeJob(files[n].Size))

	done <- 1
}

func TestSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping statistics simulation test.")
	}
	//Run a random number of jobs
	total := 100
	done := make(chan int)
	for i := 0; i < total; i++ {
		go simulate(i, done)
	}
	remaining := total
	for remaining > 0 {
		_ = <-done
		remaining--
		log.Printf("remaining: %d", remaining)
	}
	reporters.Reporters[UploadCounter].Q.Dump(os.Stdout)
	reporters.Reporters[DownloadCounter].Q.Dump(os.Stdout)
}

func logPurge(name string) {
	log.Printf("files can be removed for %s", name)
}

func init() {
	PanicOnProblem = true
	reporters = NewJobReporters(32, logPurge)
	//A set of files to download and upload
	files = []fileDescription{
		fileDescription{"chewbacca.jpg", 10234},
		fileDescription{"grumptycat.jpg", 8214},
		fileDescription{"odrive.pdf", 90234},
		fileDescription{"ConcurrencyIsNotParallelism.mp4", 1300000},
		fileDescription{"everything.doc", 28385},
	}
	jobTypes = []ReporterID{
		UploadCounter,
		DownloadCounter,
	}
}

package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"decipher.com/oduploader/autopilot"
)

var perPopulation = 40
var sleepTime = 120

func doSleep(i int) {
	zzz := rand.Intn(sleepTime)
	time.Sleep(time.Duration(zzz) * time.Second)
	//log.Printf("%d sleeps for %ds", i, zzz)
}

func doRandomAction(ap *autopilot.AutopilotContext, i int) bool {
	doSleep(i)
	r := rand.Intn(100)
	switch {
	case r > 70:
		ap.DoUpload(i, false, "")
	case r > 40:
		ap.DoDownload(i, "")
	case r > 20:
		ap.DoUpdate(i, "", "")
	case r > 10:
		return false
	}
	return true
}

func doClient(ap *autopilot.AutopilotContext, i int, clientExited chan int) {
	//log.Printf("running client %d", i)
	for {
		if doRandomAction(ap, i) == false {
			break
		}
	}
	clientExited <- i
}

func randomUploadsAndDownloads() {
	//Write output to /dev/null for this case, because the logs are very very large
	logHandle, err := os.OpenFile("/dev/null", os.O_WRONLY, 0700)
	if err != nil {
		log.Printf("Unable to start scenarion: %v", err)
		return
	}
	defer logHandle.Close()
	ap, err := autopilot.NewAutopilotContext(logHandle)
	if err != nil {
		log.Printf("Unable to start autopilot context: %v", err)
		return
	}

	clientExited := make(chan int)
	N := 20
	//Launch all clients Nx
	for n := 0; n < N; n++ {
		for i := 0; i < autopilot.Population; i++ {
			go doClient(ap, i, clientExited)
		}
	}

	//Wait for them to all exit
	stillRunning := autopilot.Population * N
	for {
		log.Printf("Waiting on %d more", stillRunning)
		i := <-clientExited
		log.Printf("Client %d exited", i)
		stillRunning--
		if stillRunning <= 0 {
			break
		}
	}
}

func main() {
	autopilot.Init()
	randomUploadsAndDownloads()
}

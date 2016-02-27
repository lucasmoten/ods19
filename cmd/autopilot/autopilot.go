package main

import (
	"decipher.com/oduploader/autopilot"
	"log"
	"time"
    "math/rand"
)

var perPopulation = 20
var sleepTime = 120

func doSleep(i int) {
	zzz := rand.Intn(sleepTime)
	time.Sleep(time.Duration(zzz) * time.Second)
	//log.Printf("%d sleeps for %ds", i, zzz)
}

func doRandomAction(i int) bool {
	doSleep(i)
	r := rand.Intn(100)
	switch {
	case r > 70:
		autopilot.DoUpload(i, false, "")
	case r > 40:
		autopilot.DoDownload(i, "")
	case r > 20:
		autopilot.DoUpdate(i, "", "")
	case r > 10:
		return false
	}
	return true
}

func doClient(i int, clientExited chan int) {
	//log.Printf("running client %d", i)
	for {
		if doRandomAction(i) == false {
			break
		}
	}
	clientExited <- i
}

func bigTest() {
	clientExited := make(chan int)
	N := 20
	//Launch all clients Nx
	for n := 0; n < N; n++ {
		for i := 0; i < autopilot.Population; i++ {
			go doClient(i, clientExited)
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
	bigTest()
}

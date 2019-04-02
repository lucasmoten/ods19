package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/client"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/util"
)

// adapted from http://tleyden.github.io/blog/2016/11/21/tuning-the-go-http-client-library-for-load-testing/

var mountPoint = util.GetClientMountPoint()

var conf = client.Config{
	Cert:       os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/test_0.cert.pem",
	Trust:      os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/server.trust.pem",
	Key:        os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/test_0.key.pem",
	SkipVerify: true,
	ServerName: config.GetEnvOrDefault("OD_PEER_CN", "twl-server-generic2"), // If you set OD_PEER_CN, then this matches it
	Remote:     mountPoint,
}

func startLoadTest() {
	count := 0

	var clients []*http.Client
	var responses []*http.Response

	// this block can be outside or inside the loop, if inside, add a break to bust out of the for
	me, err := client.NewClient(conf)
	if err != nil {
		log.Printf("could not create client: %v", err)
	}
	clients = append(clients, me.GetHttpClient())

	for {
		var err error
		var resp *http.Response

		resp, err = me.GetHttpClient().Get(fmt.Sprintf("%s/", mountPoint))
		if err != nil {
			log.Printf("Got error hitting proxier: %v", err)
			break
		}

		responses = append(responses, resp)
		var buf []byte
		resp.Body.Read(buf)
		//defer resp.Body.Close()
		log.Printf("Finished GET request #%v", count)
		count += 1
	}

	log.Printf("sleeping for a bit")
	time.Sleep(time.Duration(60) * time.Second)

}

func main() {

	log.Println("This is a bad client that retrieves a resource from server, but doesnt read the body and keeps it open")
	log.Println()
	log.Println("Make sure the docker stack is started")
	log.Println()

	startLoadTest()

}

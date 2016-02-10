package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"decipher.com/oduploader/cmd/metadataconnector/libs/server"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"
)

// ClientIdentity is a user that is going to connect to oru service
type ClientIdentity struct {
	TrustPem      string
	CertPem       string
	KeyPem        string
	Config        *tls.Config
	Name          string
	UploadCache   string
	DownloadCache string
	Index         int
}

var host = "https://dockervm:8080"

// NewClientTLSConfig creates a per-client tls config
func NewClientTLSConfig(client *ClientIdentity) (*tls.Config, error) {

	// Create the trust
	trustBytes, err := ioutil.ReadFile(client.TrustPem)
	if err != nil {
		log.Printf("Unable to read %s: %v", client.TrustPem, err)
		return nil, err
	}
	trustCertPool := x509.NewCertPool()
	if trustCertPool.AppendCertsFromPEM(trustBytes) == false {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}

	//Create certkeypair
	cert, err := tls.LoadX509KeyPair(client.CertPem, client.KeyPem)
	if err != nil {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		// Ensure that we only use our "CA" to validate certificates
		ClientCAs: trustCertPool,
		// PFS because we can but this will reject client with RSA certificates
		// CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		// Force it server side
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

func getClientIdentity(i int, name string) (*ClientIdentity, error) {
	ci := &ClientIdentity{
		TrustPem: os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/client.trust.pem"),
		CertPem:  os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/" + name + ".cert.pem"),
		KeyPem:   os.ExpandEnv("$GOPATH/src/decipher.com/oduploader/defaultcerts/clients/" + name + ".key.pem"),
	}
	cfg, err := NewClientTLSConfig(ci)
	if err != nil {
		log.Printf("Cannot get identity: %v", err)
		return nil, err
	}
	ci.Config = cfg
	ci.Name = name

	//Keep this huge directory out of $GOPATH
	if os.ExpandEnv("$AUTOPILOT_HOME") == "" {
		os.Setenv("$AUTOPILOT_HOME", os.ExpandEnv("$HOME/autopilot"))
		os.Mkdir("~/autopilot", 0700)
	}
	ci.UploadCache = os.ExpandEnv("$HOME/autopilot/uploadCache" + name)
	ci.DownloadCache = os.ExpandEnv("$HOME/autopilot/downloadCache" + name)
	ci.Index = i
	_, err = os.Stat(ci.UploadCache)
	if os.IsNotExist(err) {
		err = os.Mkdir(ci.UploadCache, 0700)
		if err != nil {
			log.Printf("Unable to make an upload cache for %s:%v", ci.UploadCache, err)
			return nil, err
		}
	}
	_, err = os.Stat(ci.DownloadCache)
	if os.IsNotExist(err) {
		err = os.Mkdir(ci.DownloadCache, 0700)
		if err != nil {
			log.Printf("Unable to make a download cache for %s:%v", name, err)
			return nil, err
		}
	}
	return ci, nil
}

var clients []*ClientIdentity

func populateClients(population int) {
	clients = make([]*ClientIdentity, population)
	for i := 0; i < len(clients); i++ {
		client, err := getClientIdentity(i, "test_"+strconv.Itoa(i))
		clients[i] = client
		if err != nil {
			log.Printf("Could not create client %d: %v", i, err)
		} else {
			//log.Printf("Creating client %d", i)
		}
	}
}

func doSleep(i int) {
	zzz := rand.Intn(sleepTime)
	time.Sleep(time.Duration(zzz) * time.Second)
	//log.Printf("%d sleeps for %ds", i, zzz)
}

func getRandomClassification() string {
	r := rand.Intn(4)
	classes := []string{"U", "C", "S", "T"}
	return classes[r]
}

func generateUploadRequest(name string, fqName string) (*http.Request, error) {
	f, err := os.Open(fqName)
	defer f.Close()
	if err != nil {
		log.Printf("Unable to open %s: %v", fqName, err)
		return nil, err
	}
	//Create a multipart mime request
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("classification", getRandomClassification())
	fw, err := w.CreateFormFile("filestream", name)
	if err != nil {
		log.Printf("unable to create form file from %s:%v", fqName, err)
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		log.Printf("Could not copy file:%v", err)
		return nil, err
	}
	w.Close()

	req, err := http.NewRequest(
		"POST",
		host+"/service/metadataconnector/1.0/object",
		&b,
	)
	if err != nil {
		log.Printf("Could not generate request: %v", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, err
}

func doUpload(i int) {
	//log.Printf("%d upload out of %s", i, clients[i].UploadCache)
	//Pick a random file
	listing, err := ioutil.ReadDir(clients[i].UploadCache)
	if err != nil {
		log.Printf("Unable to list upload directory %s", clients[i].UploadCache)
		return
	}
	if len(listing) == 0 {
		log.Printf("Nothing to upload...")
		return
	}
	//Grab a random item out of the listing (in memory... beware of huge dirs!)
	r := rand.Intn(len(listing))
	filePicked := listing[r]

	if filePicked.IsDir() == false {
		filePickedName := filePicked.Name()
		fqName := clients[i].UploadCache + "/" + filePickedName
		req, err := generateUploadRequest(filePickedName, fqName)
		if err != nil {
			log.Printf("Could not generate request:%v", err)
			return
		}

		transport := &http.Transport{TLSClientConfig: clients[i].Config}
		client := &http.Client{Transport: transport}
		res, err := client.Do(req)
		if err != nil {
			log.Printf("Error doing client request:%v", err)
			return
		}
		// Check the response
		if res.StatusCode != http.StatusOK {
			log.Printf("bad status: %s", res.Status)
			return
		}
		log.Printf("%s uploaded %s", clients[i].Name, fqName)
	}
}

func doDownload(i int) {
	//log.Printf("%d download", i)
	log.Printf("first, we must get a listing of objects to choose from")
	req, err := http.NewRequest(
		"GET",
		host+"/service/metadataconnector/1.0/objects",
		nil,
	)
	if err != nil {
		log.Printf("unable to do request for object listing:%v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	reqBytes, err := httputil.DumpRequestOut(req, true)
	log.Printf("%v\n%s", err, string(reqBytes))

	transport := &http.Transport{TLSClientConfig: clients[i].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error doing listing request:%v", err)
		return
	}
	resBytes, err := httputil.DumpResponse(res, true)
	log.Printf("%v\n%s", err, string(resBytes))

	// Check the response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		return
	}
	decoder := json.NewDecoder(res.Body)
	var listing []server.ObjectLink
	err = decoder.Decode(&listing)

	//Grab a random item (if any exist) and download it
	if len(listing) > 0 {

		//Download the listing from which to grab a random item
		randomIndex := rand.Intn(len(listing))
		link := listing[randomIndex]

		dlReq, err := http.NewRequest(
			"GET",
			host+link.URL+"/stream",
			nil,
		)
		if err != nil {
			log.Printf("Unable to generate request:%v", err)
			return
		}

		//Now download the stream into a file
		transport2 := &http.Transport{TLSClientConfig: clients[i].Config}
		client2 := &http.Client{Transport: transport2}

		dlRes, err := client2.Do(dlReq)
		if err != nil {
			log.Printf("Unable to do request:%v", err)
			return
		}
		drainFileName := clients[i].DownloadCache + "/" + link.Name
		drainFile, err := os.Create(drainFileName)
		if err != nil {
			log.Printf("Cant open %s", drainFileName)
			return
		}
		defer drainFile.Close()
		io.Copy(drainFile, dlRes.Body)
	}
}

func doRandomAction(i int) bool {
	doSleep(i)
	r := rand.Intn(100)
	switch {
	case r > 70:
		doUpload(i)
	case r > 40:
		doDownload(i)
	case r > 20:
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

var population = 10
var perPopulation = 20
var sleepTime = 240

func main() {
	flag.StringVar(&host, "url", "https://dockervm:8080", "The URL at which to direct uploads/downloads")
	flag.IntVar(&perPopulation, "perPopulation", 20, "number of uploads per user")
	flag.IntVar(&sleepTime, "sleepTime", 120, "number of seconds to sleep when we decide to sleep")
	flag.Parse()

	//We have 10 test certs (note the test_0 is known as tester10)
	population := 10
	populateClients(population)

	clientExited := make(chan int)
	N := 20
	//Launch all clients Nx
	for n := 0; n < N; n++ {
		for i := 0; i < population; i++ {
			go doClient(i, clientExited)
		}
	}

	//Wait for them to all exit
	stillRunning := population * N
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
